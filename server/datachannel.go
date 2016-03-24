package server

import (
	"bufio"
	"net"
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
	GetUserChannelFromDataChannel(id int) (
		fromUser <-chan *packet.Packet, toUser chan<- *packet.Packet, err error)
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

type DataChannel struct {
	flow    *flow.Flow
	conn    net.Conn
	session *packet.SessionIV

	heartBeat *packet.HeartBeatStage
	writeChan chan *packet.Packet
}

func NewDataChannel(f *flow.Flow, conn net.Conn, s *packet.SessionIV) (*DataChannel, error) {
	dc := &DataChannel{
		conn:      conn,
		session:   s,
		writeChan: make(chan *packet.Packet),
	}
	f.ForkTo(&dc.flow, dc.Close)
	dc.heartBeat = packet.NewHeartBeatStage(dc.flow, 3*time.Second)
	return dc, nil
}

func (d *DataChannel) GetSession() *packet.SessionIV {
	return d.session
}

// read from datachannel and write to user
func (d *DataChannel) readLoop(toUser chan<- *packet.Packet) {
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
		switch p.Type {
		case packet.HeartBeat:
			d.writeChan <- p.Reply(nil)
		case packet.HeartBeatResp:
			d.heartBeat.Receive(p.IV)
		default:
			select {
			case <-d.flow.IsClose():
				break loop
			case toUser <- p:
			}
		}
	}
}

func (d *DataChannel) write(msg *packet.Packet) error {
	_, err := d.conn.Write(msg.Marshal(d.session))
	return err
}

// read from user and write to datachannel
func (d *DataChannel) writeLoop(fromUser <-chan *packet.Packet) {
	d.flow.Add(1)
	defer d.flow.DoneAndClose()
	var err error

	heartBeatTicker := time.NewTicker(time.Second)
	defer heartBeatTicker.Stop()

loop:
	for {
		select {
		case <-d.flow.IsClose():
			break loop
		case <-heartBeatTicker.C:
			msg := d.heartBeat.New()
			err = d.write(msg)
			d.heartBeat.Add(msg.IV)
		case msg := <-fromUser:
			err = d.write(msg)
		case msg := <-d.writeChan:
			err = d.write(msg)
		}
		if err != nil {
			logex.Error(err)
			break loop
		}
	}
}

func (d *DataChannel) GetUserId() int {
	return int(d.session.UserId)
}

func (d *DataChannel) Run(
	fromUser <-chan *packet.Packet, toUser chan<- *packet.Packet) {

	go d.writeLoop(fromUser)
	go d.readLoop(toUser)
}

func (d *DataChannel) Close() {
	d.conn.Close()
	d.flow.Close()
}
