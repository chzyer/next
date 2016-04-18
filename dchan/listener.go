package dchan

import (
	"net"
	"strconv"
	"strings"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"gopkg.in/logex.v1"
)

// add self monitor
type Listener struct {
	ln       net.Listener
	flow     *flow.Flow
	delegate SvrDelegate
	port     int
	onClose  func()
}

func NewListener(f *flow.Flow, d SvrDelegate, c func()) (*Listener, error) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}
	addr := ln.Addr().String()
	if idx := strings.LastIndex(addr, ":"); idx > 0 {
		addr = addr[idx+1:]
	}
	port, err := strconv.Atoi(addr)
	if err != nil {
		panic(err)
	}
	dcln := &Listener{
		ln:       ln,
		port:     port,
		delegate: d,
		onClose:  c,
	}
	f.ForkTo(&dcln.flow, dcln.Close)
	return dcln, nil
}

func (d *Listener) GetPort() int {
	return d.port
}

func (d *Listener) Accept() (Channel, error) {
	conn, err := d.ln.Accept()
	if err != nil {
		return nil, logex.Trace(err)
	}
	session, err := d.checkAuth(conn)
	if err != nil {
		return nil, logex.Trace(err)
	}
	_, toUser, err := d.delegate.GetUserChannelFromDataChannel(int(session.UserId))
	if err != nil {
		return nil, logex.Trace(err)
	}

	ch := NewTcpChan(d.flow, session, conn, toUser)
	return ch, nil
}

func (d *Listener) Serve() {
	d.flow.Add(1)
	defer d.flow.DoneAndClose()

	for !d.flow.IsClosed() {
		ch, err := d.Accept()
		if err != nil {
			break
		}
		d.delegate.OnNewChannel(ch)
		go ch.Run()
	}
}

func (d *Listener) Close() {
	if !d.flow.MarkExit() {
		return
	}
	logex.Info("listener:", d.port, "closed")
	d.ln.Close()
	d.flow.Close()
	d.onClose()
}

func (d *Listener) checkAuth(conn net.Conn) (*packet.SessionIV, error) {
	return ServerCheckAuth(d.delegate, d.port, conn)
}
