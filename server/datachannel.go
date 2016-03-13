package server

import (
	"bufio"
	"bytes"
	"fmt"
	"net"

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
}

type DataChannel struct {
	ln       net.Listener
	flow     *flow.Flow
	port     int
	delegate DataChannelDelegate
	in       chan *packet.Packet
	out      chan *packet.Packet
}

func NewDataChannel(port int, f *flow.Flow, d DataChannelDelegate,
	in, out chan *packet.Packet) (*DataChannel, error) {

	ln, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		return nil, err
	}
	dc := &DataChannel{
		port:     port,
		ln:       ln,
		delegate: d,
		flow:     f,
		in:       in,
		out:      out,
	}
	f.SetOnClose(dc.Close)
	return dc, nil
}

func (d *DataChannel) readLoop(f *flow.Flow, s *packet.SessionIV, conn net.Conn) {
	f.Add(1)
	defer f.DoneAndClose()
	buf := bufio.NewReader(conn)
loop:
	for {
		p, err := packet.Read(s, buf)
		if err != nil {
			break
		}
		select {
		case <-f.IsClose():
			break loop
		case d.out <- p:
		}
	}
}

func (d *DataChannel) writeLoop(f *flow.Flow, s *packet.SessionIV, conn net.Conn) {
	f.Add(1)
	defer f.DoneAndClose()
loop:
	for {
		select {
		case <-f.IsClose():
			break loop
		case msg := <-d.in:
			_, err := conn.Write(msg.Marshal(s))
			if err != nil {
				logex.Error(err)
				break loop
			}
		}
	}
}

func (d *DataChannel) checkAuth(conn net.Conn) (*packet.SessionIV, error) {
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

	p, err = packet.New(s.Token, packet.AuthResp)
	if err != nil {
		panic(err)
	}
	if _, err := conn.Write(p.Marshal(s)); err != nil {
		return nil, logex.Trace(err)
	}
	return s, nil
}

func (d *DataChannel) loop() {
	d.flow.Add(1)
	defer d.flow.DoneAndClose()

	for {
		conn, err := d.ln.Accept()
		if err != nil {
			logex.Error(err)
			return
		}
		session, err := d.checkAuth(conn)
		if err != nil {
			logex.Error(err)
			conn.Close()
			continue
		}
		f := d.flow.Fork(0)
		f.SetOnClose(func() {
			conn.Close()
		})

		go d.readLoop(f, session, conn)
		go d.writeLoop(f, session, conn)
	}
}

func (d *DataChannel) Close() {
	d.ln.Close()
	d.flow.Close()
}
