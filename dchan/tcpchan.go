package dchan

import (
	"bufio"
	"fmt"
	"net"
	"strings"
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
		in:    make(chan *packet.Packet),
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(false)
	}

	f.ForkTo(&ch.flow, ch.Close)
	ch.heartBeat = statistic.NewHeartBeatStage(ch.flow, 5*time.Second, ch)
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
	c.exitError = logex.NewErrorf("clean: %v", err)
	c.Close()
}

func (c *TcpChan) Run() {
	go c.writeLoop()
	go c.readLoop()
}

func (c *TcpChan) rawWrite(p []*packet.Packet) error {
	l2 := packet.WrapL2(c.session, p)
	data := c.WriteL2(l2)
	n, err := c.conn.Write(data)
	c.speed.Upload(n)
	return err
}

func (c *TcpChan) writeLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

	if !c.flow.WaitNotify(c.waitInitChan) {
		return
	}

	bufTimer := time.NewTimer(time.Millisecond)

	heartBeatTicker := time.NewTicker(1 * time.Second)
	defer heartBeatTicker.Stop()

	var bufferingPackets []*packet.Packet
	var err error
loop:
	for {
		select {
		case <-c.flow.IsClose():
			break loop
		case <-heartBeatTicker.C:
			p := c.heartBeat.New()
			err = c.rawWrite([]*packet.Packet{p})
			c.heartBeat.Add(p)
		case p := <-c.in:
			bufTimer.Reset(time.Millisecond)
			bufferingPackets = append(bufferingPackets, p)
		buffering:
			for {
				select {
				case <-bufTimer.C:
					break buffering
				case p := <-c.in:
					bufferingPackets = append(bufferingPackets, p)
				}
			}
			err = c.rawWrite(bufferingPackets)
			bufferingPackets = bufferingPackets[:0]
		}
		if err != nil {
			if !strings.Contains(err.Error(), "closed") {
				c.exitError = logex.NewErrorf("write error: %v", err)
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
		c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		l2, err := c.ReadL2(buf)
		if err != nil {
			if err, ok := err.(*net.OpError); ok {
				if err.Temporary() || err.Timeout() {
					continue
				}
			}
			if !strings.Contains(err.Error(), "closed") {
				c.exitError = logex.NewErrorf("read error: %v", err)
			}
			break
		}

		if err := l2.Verify(c.session); err != nil {
			c.exitError = logex.NewErrorf("verify error: %v", err)
			break
		}

		if c.IsSvrModeAndUninit() {
			out, err := c.delegate.Init(int(l2.UserId))
			if err != nil {
				c.exitError = logex.NewErrorf("init error: %v", err)
				break
			}
			c.markInit(out)
			c.delegate.OnInited(c)
		}

		ps, err := l2.Unmarshal()
		if err != nil {
			c.exitError = logex.NewErrorf("packet error: %v", err)
			break
		}

		for _, p := range ps {
			c.speed.Download(p.Size())
			if !c.onRecePacket(p) {
				break loop
			}
		}
	}
}

func (h *TcpChan) onRecePacket(p *packet.Packet) bool {
	switch p.Type {
	case packet.HEARTBEAT:
		select {
		case h.in <- p.Reply(nil):
		case <-h.flow.IsClose():
			return false
		}
	case packet.HEARTBEAT_R:
		h.heartBeat.Receive(p)
	default:
		select {
		case <-h.flow.IsClose():
			return false
		case h.out <- p:
		}
	}
	return true
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

func (c *TcpChan) Name() string {
	return fmt.Sprintf("[%v -> %v]",
		c.conn.LocalAddr(),
		c.conn.RemoteAddr(),
	)
}

func (c *TcpChan) GetStat() *statistic.HeartBeat {
	return c.heartBeat.GetStat()
}
