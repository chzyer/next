package datachannel

import (
	"bufio"
	"fmt"
	"net"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"gopkg.in/logex.v1"
)

type DC struct {
	cfg       *Config
	flow      *flow.Flow
	session   *packet.SessionIV
	conn      net.Conn
	writeChan chan *packet.Packet

	heartBeat *packet.HeartBeatStage
}

type Config struct {
	OnClose func()
}

func New(f *flow.Flow, conn net.Conn, session *packet.SessionIV, cfg *Config) *DC {

	dc := &DC{
		session:   session,
		conn:      conn,
		cfg:       cfg,
		writeChan: make(chan *packet.Packet, 4),
	}

	f.ForkTo(&dc.flow, dc.Close)
	dc.heartBeat = packet.NewHeartBeatStage(dc.flow, 3*time.Second)
	return dc
}

func (d *DC) Run(in <-chan *packet.Packet, out chan<- *packet.Packet) {

	go d.writeLoop(in)
	go d.readLoop(out)
}

func (d *DC) readLoop(out chan<- *packet.Packet) {
	d.flow.Add(1)
	defer d.flow.DoneAndClose()

	buf := bufio.NewReader(d.conn)
loop:
	for {
		p, err := packet.Read(d.session, buf)
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
			case out <- p:
			}
		}
	}
}

func (d *DC) write(p *packet.Packet) error {
	_, err := d.conn.Write(p.Marshal(d.session))
	return err
}

func (d *DC) writeLoop(in <-chan *packet.Packet) {
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
		case p := <-in:
			err = d.write(p)
		}
		if err != nil {
			logex.Error(err)
			break
		}
	}
}

func (d *DC) GetStat() *packet.HeartBeatStat {
	return d.heartBeat.GetStat()
}

func (d *DC) Name() string {
	return fmt.Sprintf("[%v -> %v]",
		d.conn.LocalAddr(),
		d.conn.RemoteAddr(),
	)
}

func (d *DC) Close() {
	logex.Info(d.Name(), "close")
	d.conn.Close()
	d.flow.Close()
	if d.cfg.OnClose != nil {
		d.cfg.OnClose()
	}
}

func (d *DC) GetUserId() int {
	return int(d.session.UserId)
}

func (d *DC) GetSession() *packet.SessionIV {
	return d.session
}

func DialDC(host string, f *flow.Flow, session *packet.SessionIV,
	onClose func(), in, out chan *packet.Packet) (*DC, error) {

	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, logex.Trace(err)
	}
	if err := ClientCheckAuth(conn, session); err != nil {
		return nil, logex.Trace(err)
	}
	dc := New(f, conn, session, &Config{
		OnClose: onClose,
	})
	dc.Run(in, out)
	return dc, nil
}
