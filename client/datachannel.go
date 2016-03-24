package client

import (
	"bufio"
	"bytes"
	"net"
	"time"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

type DataChannel struct {
	s         *packet.SessionIV
	conn      net.Conn
	flow      *flow.Flow
	in        chan *packet.Packet
	out       chan *packet.Packet
	writeChan chan *packet.Packet
	onClose   func()

	heartBeat *packet.HeartBeatStage
}

func NewDataChannel(host string, f *flow.Flow, session *packet.SessionIV,
	onClose func(), in, out chan *packet.Packet) (*DataChannel, error) {

	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, logex.Trace(err)
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
		s:         session,
		conn:      conn,
		in:        in,
		out:       out,
		onClose:   onClose,
		writeChan: make(chan *packet.Packet, 4),
	}
	f.ForkTo(&dc.flow, dc.Close)
	dc.heartBeat = packet.NewHeartBeatStage(dc.flow, 3*time.Second)

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
		switch p.Type {
		case packet.HeartBeat:
			d.writeChan <- p.Reply(nil)
		case packet.HeartBeatResp:
			d.heartBeat.Receive(p.IV)
		default:
			select {
			case <-d.flow.IsClose():
				break loop
			case d.out <- p:
			}
		}
	}
}

func (d *DataChannel) write(p *packet.Packet) error {
	_, err := d.conn.Write(p.Marshal(d.s))
	return err
}

func (d *DataChannel) writeLoop() {
	d.flow.Add(1)
	defer d.flow.DoneAndClose()
	heartBeatTicker := time.NewTicker(time.Second)
	defer heartBeatTicker.Stop()

	var err error
loop:
	for {
		select {
		case <-d.flow.IsClose():
			break loop
		case <-heartBeatTicker.C:
			p := d.heartBeat.New()
			err = d.write(p)
			d.heartBeat.Add(p.IV)
		case p := <-d.writeChan:
			err = d.write(p)
		case p := <-d.in:
			err = d.write(p)
		}
		if err != nil {
			logex.Error(err)
			break
		}
	}
}

func (d *DataChannel) Close() {
	d.conn.Close()
	d.flow.Close()
	if d.onClose != nil {
		d.onClose()
	}
}
