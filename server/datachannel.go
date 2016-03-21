package server

import (
	"bufio"
	"bytes"
	"net"
	"strconv"
	"strings"
	"time"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

var (
	ErrInvalidUserId        = logex.Define("invalid user id")
	ErrUnexpectedPacketType = logex.Define("unexpected packet type")
)

type DataChannelDelegate interface {
	GetUserToken(id int) string
	GetUserChannel(id int) (in, out chan *packet.Packet, err error)
}

type MultiDataChannel struct {
	flow      *flow.Flow
	delegate  DataChannelDelegate
	listeners []*DataChannelListener
}

func NewMultiDataChannel(f *flow.Flow, d DataChannelDelegate) *MultiDataChannel {
	m := &MultiDataChannel{
		flow:     f,
		delegate: d,
	}
	return m
}

func (m *MultiDataChannel) GetDataChannel() int {
	return m.listeners[0].GetPort()
}

func (m *MultiDataChannel) Start(n int) {
	m.flow.Add(1)
	defer m.flow.DoneAndClose()

	started := 0
loop:
	for !m.flow.IsClosed() {
		if started < n {
			m.AddChannelListen()
			started++
		} else {
			select {
			case <-m.flow.IsClose():
				break loop
			case <-time.After(time.Second):
			}
		}
	}
}

func (m *MultiDataChannel) AddChannelListen() error {
	ln, err := NewDataChannelListener(m.flow, m.delegate)
	if err != nil {
		return logex.Trace(err)
	}
	m.listeners = append(m.listeners, ln)

	go ln.Serve()
	return nil
}

// -----------------------------------------------------------------------------

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
	_, _, err = d.delegate.GetUserChannel(int(session.UserId))
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
		in, out, _ := d.delegate.GetUserChannel(dc.GetUserId())
		go dc.Run(in, out)
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

// -----------------------------------------------------------------------------

type DataChannel struct {
	flow    *flow.Flow
	conn    net.Conn
	session *packet.SessionIV
}

func NewDataChannel(f *flow.Flow, conn net.Conn, s *packet.SessionIV) (*DataChannel, error) {
	dc := &DataChannel{
		conn:    conn,
		session: s,
	}
	f.ForkTo(&dc.flow, dc.Close)
	return dc, nil
}

func (d *DataChannel) GetSession() *packet.SessionIV {
	return d.session
}

func (d *DataChannel) readLoop(out chan *packet.Packet) {
	d.flow.Add(1)
	defer d.flow.DoneAndClose()
	buf := bufio.NewReader(d.conn)
loop:
	for {
		p, err := packet.Read(d.session, buf)
		if err != nil {
			logex.Error(err)
			break
		}
		select {
		case <-d.flow.IsClose():
			break loop
		case out <- p:
			logex.Info(p)
		}
	}
}

func (d *DataChannel) writeLoop(in chan *packet.Packet) {
	d.flow.Add(1)
	defer d.flow.DoneAndClose()
loop:
	for {
		select {
		case <-d.flow.IsClose():
			break loop
		case msg := <-in:
			_, err := d.conn.Write(msg.Marshal(d.session))
			if err != nil {
				logex.Error(err)
				break loop
			}
		}
	}
}

func (d *DataChannel) GetUserId() int {
	return int(d.session.UserId)
}

func (d *DataChannel) Run(in, out chan *packet.Packet) {
	go d.writeLoop(out)
	go d.readLoop(in)
}

func (d *DataChannel) Close() {
	d.conn.Close()
	d.flow.Close()
}
