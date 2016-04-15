package dchan

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/statistic"
	"gopkg.in/logex.v1"
)

type TcpChan struct {
	flow    *flow.Flow
	session *packet.SessionIV
	conn    net.Conn

	// private
	heartBeat *packet.HeartBeatStage
	speed     *statistic.Speed

	// runtime
	exitError error

	in  chan *packet.Packet
	out chan<- *packet.Packet
}

func NewTcpChan(f *flow.Flow, session *packet.SessionIV, conn net.Conn, out chan<- *packet.Packet) *TcpChan {
	ch := &TcpChan{
		session: session,
		conn:    conn,

		speed: statistic.NewSpeed(),
		in:    make(chan *packet.Packet, 8),
		out:   out,
	}
	f.ForkTo(&ch.flow, ch.Close)
	ch.heartBeat = packet.NewHeartBeatStage(ch.flow, 5*time.Second, ch)
	return ch
}

func (c *TcpChan) GetSpeed() *statistic.SpeedInfo {
	return c.speed.GetSpeed()
}

func (c *TcpChan) HeartBeatClean(err error) {
	c.exitError = fmt.Errorf("clean: %v", err)
	c.Close()
}

func (c *TcpChan) Run() {
	go c.writeLoop()
	go c.readLoop()
}

func (c *TcpChan) rawWrite(p *packet.Packet) error {
	n, err := c.conn.Write(p.Marshal(c.session))
	c.speed.Upload(n)
	return err
}

func (c *TcpChan) writeLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

	heartBeatTicker := time.NewTicker(1 * time.Second)
	defer heartBeatTicker.Stop()

	var err error
loop:
	for {
		select {
		case <-c.flow.IsClose():
			break loop
		case <-heartBeatTicker.C:
			p := c.heartBeat.New()
			err = c.rawWrite(p)
			c.heartBeat.Add(p.IV)
		case p := <-c.in:
			err = c.rawWrite(p)
		}
		if err != nil {
			if !strings.Contains(err.Error(), "closed") {
				c.exitError = fmt.Errorf("write error: %v", err)
			}
			break
		}
	}
}

func (c *TcpChan) readLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

	buf := bufio.NewReader(c.conn)
loop:
	for !c.flow.IsClosed() {
		c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		p, err := packet.Read(c.session, buf)
		if err != nil {
			if !strings.Contains(err.Error(), "closed") {
				c.exitError = fmt.Errorf("read error: %v", err)
			}
			break
		}
		c.speed.Download(p.Size())
		switch p.Type {
		case packet.HEARTBEAT:
			select {
			case c.in <- p.Reply(nil):
			case <-c.flow.IsClose():
				break loop
			}
		case packet.HEARTBEAT_R:
			c.heartBeat.Receive(p.IV)
		default:
			select {
			case <-c.flow.IsClose():
				break loop
			case c.out <- p:
			}
		}
	}
}

func (c *TcpChan) Latency() (latency, lastCommit time.Duration) {
	return c.heartBeat.GetLatency()
}

func (c *TcpChan) ChanWrite() chan<- *packet.Packet {
	return c.in
}

func (c *TcpChan) AddOnClose(f func()) {
	c.flow.AddOnClose(f)
}

func (c *TcpChan) Close() {
	if !c.flow.MarkExit() {
		return
	}
	if c.exitError != nil {
		logex.DownLevel(1).Debug("where is exit")
		logex.Info(c.Name(), "exit by:", c.exitError)
	} else {
		logex.Info(c.Name(), "exit manually")
	}
	c.conn.Close()
	c.flow.Close()
}

func (c *TcpChan) GetUserId() int {
	return int(c.session.UserId)
}

func (c *TcpChan) Src() net.Addr {
	return c.conn.LocalAddr()
}

func (c *TcpChan) Dst() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *TcpChan) Name() string {
	return fmt.Sprintf("[%v -> %v]",
		c.conn.LocalAddr(),
		c.conn.RemoteAddr(),
	)
}

func (c *TcpChan) GetStat() *packet.HeartBeatStat {
	return c.heartBeat.GetStat()
}
