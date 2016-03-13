package client

import (
	"bufio"
	"bytes"
	"fmt"
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

	conn, err := net.Dial("tcp", fmt.Sprintf("%v:%v", host, session.Port))
	if err != nil {
		return nil, err
	}
	dc := &DataChannel{
		s:    session,
		conn: conn,
		flow: f,
		in:   in,
		out:  out,
	}
	f.SetOnClose(dc.Close)

	p, err := packet.New(session.Token, packet.Auth)
	if err != nil {
		return nil, logex.Trace(err)
	}
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
				break loop
			}
		}
	}
}

func (d *DataChannel) Close() {
	d.conn.Close()
	d.flow.Close()
}
