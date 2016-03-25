package datachannel

import (
	"net"
	"strconv"
	"strings"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"gopkg.in/logex.v1"
)

type Listener struct {
	ln       net.Listener
	flow     *flow.Flow
	delegate SvrDelegate
	port     int
}

func NewListener(f *flow.Flow, d SvrDelegate) (*Listener, error) {
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
	}
	f.ForkTo(&dcln.flow, dcln.Close)
	return dcln, nil
}

func (d *Listener) GetPort() int {
	return d.port
}

func (d *Listener) Accept() (*DC, error) {
	conn, err := d.ln.Accept()
	if err != nil {
		return nil, logex.Trace(err)
	}
	session, err := d.checkAuth(conn)
	if err != nil {
		return nil, logex.Trace(err)
	}
	_, _, err = d.delegate.GetUserChannelFromDataChannel(int(session.UserId))
	if err != nil {
		return nil, logex.Trace(err)
	}

	dc := New(d.flow, conn, session, &Config{})
	return dc, nil
}

func (d *Listener) Serve() {
	d.flow.Add(1)
	defer d.flow.DoneAndClose()

	for !d.flow.IsClosed() {
		dc, err := d.Accept()
		if err != nil {
			break
		}
		fromUser, toUser, _ := d.delegate.GetUserChannelFromDataChannel(
			dc.GetUserId())
		go dc.Run(fromUser, toUser)
	}
}

func (d *Listener) Close() {
	d.ln.Close()
	d.flow.Close()
}

func (d *Listener) checkAuth(conn net.Conn) (*packet.SessionIV, error) {
	return ServerCheckAuth(d.delegate, d.port, conn)
}
