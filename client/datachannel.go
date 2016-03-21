package client

import (
	"bufio"
	"bytes"
	"net"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

type DataChannel struct {
	s    *packet.SessionIV
	conn net.Conn
	flow *flow.Flow
	in   chan *packet.Packet
	out  chan *packet.Packet
}

func NewDataChannel(host string, f *flow.Flow, session *packet.SessionIV,
	in, out chan *packet.Packet) (*DataChannel, error) {

	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	p := packet.New(session.Token, packet.Auth)
	if _, err = conn.Write(p.Marshal(session)); err != nil {
		return nil, logex.Trace(err)
	}
	pr, err := packet.Read(session, conn)
	if err != nil {
		return nil, logex.Trace(err)
	}
	if !bytes.Equal(pr.Payload, p.Payload) {
		return nil, logex.NewError("invalid auth reply", pr.Payload, p.Payload)
	}

	dc := &DataChannel{
		s:    session,
		conn: conn,
		in:   in,
		out:  out,
	}
	f.ForkTo(&dc.flow, dc.Close)

	go dc.readLoop()
	go dc.writeLoop()

	return dc, nil
}

func (d *DataChannel) readLoop() {
	d.flow.Add(1)
	defer d.flow.DoneAndClose()

	buf := bufio.NewReader(d.conn)
loop:
	for {
		p, err := packet.Read(d.s, buf)
		if err != nil {
			break
		}
		select {
		case <-d.flow.IsClose():
			break loop
		case d.out <- p:
		}
	}
}

func (d *DataChannel) writeLoop() {
	d.flow.Add(1)
	defer d.flow.DoneAndClose()

loop:
	for {
		select {
		case <-d.flow.IsClose():
			break loop
		case p := <-d.in:
			_, err := d.conn.Write(p.Marshal(d.s))
			if err != nil {
				logex.Error(err)
				break loop
			}
		}
	}
}

func (d *DataChannel) Close() {
	d.conn.Close()
	d.flow.Close()
}
