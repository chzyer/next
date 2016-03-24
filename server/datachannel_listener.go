package server

import (
	"bytes"
	"net"
	"strconv"
	"strings"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"gopkg.in/logex.v1"
)

type DataChannelListener struct {
	ln       net.Listener
	flow     *flow.Flow
	delegate DataChannelDelegate
	port     int
}

func NewDataChannelListener(f *flow.Flow, d DataChannelDelegate) (*DataChannelListener, error) {
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
	dcln := &DataChannelListener{
		ln:       ln,
		port:     port,
		delegate: d,
	}
	f.ForkTo(&dcln.flow, dcln.Close)
	return dcln, nil
}

func (d *DataChannelListener) GetPort() int {
	return d.port
}

func (d *DataChannelListener) Accept() (*DataChannel, error) {
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

	dc, err := NewDataChannel(d.flow, conn, session)
	if err != nil {
		return nil, logex.Trace(err)
	}

	return dc, nil
}

func (d *DataChannelListener) Serve() {
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

func (d *DataChannelListener) Close() {
	d.ln.Close()
	d.flow.Close()
}

func (d *DataChannelListener) checkAuth(conn net.Conn) (*packet.SessionIV, error) {
	iv, err := packet.ReadIV(conn)
	if err != nil {
		return nil, logex.Trace(err)
	}
	if int(iv.Port) != d.port {
		return nil, packet.ErrPortNotMatch.Trace()
	}

	token := d.delegate.GetUserToken(int(iv.UserId))
	if token == "" {
		return nil, ErrInvalidUserId.Trace()
	}

	s := packet.NewSessionIV(iv.UserId, iv.Port, []byte(token))
	p, err := packet.ReadWithIV(s, iv, conn)
	if err != nil {
		return nil, logex.Trace(err)
	}
	if p.Type != packet.Auth {
		return nil, ErrUnexpectedPacketType.Trace()
	}
	if !bytes.Equal(s.Token, p.Payload) {
		return nil, packet.ErrInvalidToken.Trace()
	}

	p = packet.New(s.Token, packet.AuthResp)
	if _, err := conn.Write(p.Marshal(s)); err != nil {
		return nil, logex.Trace(err)
	}
	return s, nil
}
