package dchan

import (
	"bufio"
	"fmt"
	"net"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/logex"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/statistic"
)

var (
	_ Channel        = new(TcpChan)
	_ ChannelFactory = new(TcpChanFactory)
)

type TcpChan struct {
	flow    *flow.Flow
	session *packet.Session
	conn    net.Conn

	delegate     SvrInitDelegate
	waitInitChan chan struct{}

	// private
	heartBeat *statistic.HeartBeatStage
	speed     *statistic.Speed

	// runtime
	exitError error

	in  chan *packet.Packet
	out chan<- *packet.Packet
}

func NewTcpChanClient(f *flow.Flow, session *packet.Session, conn net.Conn, out chan<- *packet.Packet) Channel {
	ch := NewTcpChanServer(f, session, conn, nil).(*TcpChan)
	ch.markInit(out)
	return ch
}

func NewTcpChanServer(f *flow.Flow, session *packet.Session, conn net.Conn, delegate SvrInitDelegate) Channel {
	ch := &TcpChan{
		conn:         conn,
		delegate:     delegate,
		session:      session,
		waitInitChan: make(chan struct{}, 1),

		speed: statistic.NewSpeed(),
		in:    make(chan *packet.Packet, 8),
	}
	f.ForkTo(&ch.flow, ch.Close)
	ch.heartBeat = statistic.NewHeartBeatStage(ch.flow, time.Second, ch)
	return ch
}

func (c *TcpChan) markInit(out chan<- *packet.Packet) {
	c.out = out
	c.waitInitChan <- struct{}{}
}

func (c *TcpChan) IsSvrModeAndUninit() bool {
	return c.out == nil
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
	l2 := packet.WrapL2(c.session, p)
	n, err := c.conn.Write(c.MarshalL2(l2))
	c.speed.Upload(n)
	return err
}

func (c *TcpChan) writeLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

	if !c.flow.WaitNotify(c.waitInitChan) {
		return
	}

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
			c.heartBeat.Add(p)
		case p := <-c.in:
			err = c.rawWrite(p)
		}
		if err != nil {
			c.exitError = fmt.Errorf("write error: %v", err)
			break
		}
	}
}

func (TcpChan) Factory() ChannelFactory {
	return TcpChanFactory{}
}

func (c *TcpChan) readLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

	buf := bufio.NewReader(c.conn)
loop:
	for !c.flow.IsClosed() {
		c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		l2, err := c.ReadL2(buf)
		if err != nil {
			c.exitError = fmt.Errorf("read error: %v", err)
			break
		}
		if err := l2.Verify(c.session); err != nil {
			c.exitError = fmt.Errorf("verify error: %v", err)
			break
		}

		if c.IsSvrModeAndUninit() {
			out, err := c.delegate.Init(int(l2.UserId))
			if err != nil {
				c.exitError = fmt.Errorf("init error: %v", err)
				break
			}
			c.markInit(out)
			c.delegate.OnInited(c)
		}

		p, err := packet.Unmarshal(l2.Payload)
		if err != nil {
			c.exitError = fmt.Errorf("packet error: %v", err)
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
			c.heartBeat.Receive(p)
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
		logex.Info(c.Name(), "exit by:", c.exitError)
	} else {
		logex.Info(c.Name(), "exit manually")
	}
	c.conn.Close()
	c.flow.Close()
}

func (c *TcpChan) GetUserId() (int, error) {
	return c.session.UserId(), nil
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

func (c *TcpChan) GetStat() *statistic.HeartBeat {
	return c.heartBeat.GetStat()
}
