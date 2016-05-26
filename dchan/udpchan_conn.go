package dchan

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/chzyer/flow"
)

var _ net.Listener = new(UDPListener)

type UDPListener struct {
	flow *flow.Flow
	conn *net.UDPConn

	acceptChan chan *UDPConn
	closeChan  chan string
	conns      map[string]chan []byte

	writeGuard sync.Mutex
	connGuard  sync.Mutex
}

func NewUDPListener(f *flow.Flow, conn *net.UDPConn) *UDPListener {
	ln := &UDPListener{
		conn:       conn,
		acceptChan: make(chan *UDPConn),
		closeChan:  make(chan string),
		conns:      make(map[string]chan []byte),
	}
	f.ForkTo(&ln.flow, func() {
		ln.Close()
	})
	go ln.loop()
	return ln
}

func (u *UDPListener) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	u.writeGuard.Lock()
	n, err := u.conn.WriteToUDP(b, addr)
	u.writeGuard.Unlock()
	return n, err
}

func (u *UDPListener) newConn(addr *net.UDPAddr) *UDPConn {
	conn := NewUDPConn(u.flow, u.conn.LocalAddr().(*net.UDPAddr), addr, u)
	u.acceptChan <- conn
	return conn
}

func (u *UDPListener) write(addr *net.UDPAddr, b []byte) {
	u.connGuard.Lock()
	outGoing, ok := u.conns[addr.String()]
	if !ok {
		conn := u.newConn(addr)
		outGoing = conn.WriteChan()
		u.conns[addr.String()] = outGoing
	}
	u.connGuard.Unlock()
	select {
	case outGoing <- b:
	case <-u.flow.IsClose():
	}
}

func (u *UDPListener) loop() {
	u.flow.Add(1)
	defer u.flow.DoneAndClose()

	buf := make([]byte, 64<<10)
loop:
	for !u.flow.IsClosed() {
		n, addr, err := u.conn.ReadFromUDP(buf)
		if err != nil {
			break loop
		}
		u.write(addr, buf[:n])
	}
}

func (u *UDPListener) Accept() (net.Conn, error) {
	select {
	case conn := <-u.acceptChan:
		return conn, nil
	case <-u.flow.IsClose():
		return nil, fmt.Errorf("listen on closed network")
	}
}

func (u *UDPListener) Addr() net.Addr {
	return u.conn.LocalAddr()
}

func (u *UDPListener) ClientClose(addr *net.UDPAddr) {
	u.connGuard.Lock()
	delete(u.conns, addr.String())
	u.connGuard.Lock()
}

func (u *UDPListener) Close() error {
	if !u.flow.MarkExit() {
		return nil
	}
	u.conn.Close()
	u.flow.Close()
	return nil
}

var _ net.Conn = new(UDPConn)

type writeDelegate interface {
	WriteToUDP([]byte, *net.UDPAddr) (int, error)
	ClientClose(*net.UDPAddr)
}

type UDPConn struct {
	flow       *flow.Flow
	in         chan []byte
	localAddr  *net.UDPAddr
	remoteAddr *net.UDPAddr
	delegate   writeDelegate

	rdead *time.Timer
	wdead *time.Timer
}

func NewUDPConn(f *flow.Flow, local, remote *net.UDPAddr, delegate writeDelegate) *UDPConn {
	conn := &UDPConn{
		in:         make(chan []byte, 1),
		localAddr:  local,
		remoteAddr: remote,
		delegate:   delegate,

		rdead: time.NewTimer(0),
		wdead: time.NewTimer(0),
	}
	conn.SetDeadline(time.Unix(0, 0))

	f.ForkTo(&conn.flow, func() {
		conn.Close()
	})
	return conn
}

func (u *UDPConn) LocalAddr() net.Addr {
	return u.localAddr
}

func (u *UDPConn) SetDeadline(d time.Time) error {
	u.SetReadDeadline(d)
	u.SetWriteDeadline(d)
	return nil
}

func (u *UDPConn) SetReadDeadline(d time.Time) error {
	return u.setDeadline(u.rdead, d)
}

func (u *UDPConn) SetWriteDeadline(d time.Time) error {
	return u.setDeadline(u.wdead, d)
}

func (u *UDPConn) setDeadline(t *time.Timer, d time.Time) error {
	if d.IsZero() {
		t.Stop()
		return nil
	}
	t.Reset(d.Sub(time.Now()))
	return nil
}

func (u *UDPConn) RemoteAddr() net.Addr {
	return u.remoteAddr
}

func (u *UDPConn) Read(b []byte) (int, error) {
	select {
	case buf := <-u.in:
		n := copy(b, buf)
		return n, nil
	case <-u.flow.IsClose():
		return 0, fmt.Errorf("conn is closed")
	}
}

func (u *UDPConn) Write(b []byte) (int, error) {
	return u.delegate.WriteToUDP(b, u.remoteAddr)
}

func (u *UDPConn) WriteChan() chan []byte {
	return u.in
}

func (u *UDPConn) Close() error {
	if !u.flow.MarkExit() {
		return nil
	}
	close(u.in)
	u.flow.Close()
	u.delegate.ClientClose(u.remoteAddr)
	return nil
}
