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
	_ Channel        = new(HttpChan)
	_ ChannelFactory = new(HttpChanFactory)
)

// to simulate http interactive
type HttpChan struct {
	flow         *flow.Flow
	session      *packet.Session
	in           chan *packet.Packet
	out          chan<- *packet.Packet
	onClose      func()
	conn         net.Conn
	waitInitChan chan struct{}
	delegate     SvrInitDelegate

	heartBeat *statistic.HeartBeatStage
	speed     *statistic.Speed
	exitError error
}

func NewHttpChanClient(f *flow.Flow, session *packet.Session, conn net.Conn, out chan<- *packet.Packet) *HttpChan {
	hc := NewHttpChanServer(f, session, conn, nil)
	hc.markInit(out)
	return hc
}

func NewHttpChanServer(f *flow.Flow, s *packet.Session, conn net.Conn, delegate SvrInitDelegate) *HttpChan {
	hc := &HttpChan{
		session:      s,
		conn:         conn,
		speed:        statistic.NewSpeed(),
		delegate:     delegate,
		in:           make(chan *packet.Packet),
		waitInitChan: make(chan struct{}, 1),
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(false)
	}
	f.ForkTo(&hc.flow, hc.Close)
	hc.heartBeat = statistic.NewHeartBeatStage(hc.flow, 5*time.Second, hc)
	return hc
}

func (c *HttpChan) markInit(out chan<- *packet.Packet) {
	c.out = out
	c.waitInitChan <- struct{}{}
}

func (h *HttpChan) IsSvrModeAndUninit() bool {
	return h.out == nil
}

func (h *HttpChan) HeartBeatClean(err error) {
	h.exitError = logex.NewErrorf("clean: %v", err)
	h.Close()
}

func (h *HttpChan) AddOnClose(f func()) {
	h.flow.AddOnClose(f)
}

func (h *HttpChan) rawWrite(p []*packet.Packet) error {
	l2 := packet.WrapL2(h.session, p)
	n, err := h.conn.Write(h.WriteL2(l2))
	h.speed.Upload(n)
	return err
}

func (h *HttpChan) writeLoop() {
	h.flow.Add(1)
	defer h.flow.DoneAndClose()

	if !h.flow.WaitNotify(h.waitInitChan) {
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
		case <-h.flow.IsClose():
			break loop
		case <-heartBeatTicker.C:
			p := h.heartBeat.New()
			err = h.rawWrite([]*packet.Packet{p})
			h.heartBeat.Add(p)
		case p := <-h.in:
			bufTimer.Reset(time.Millisecond)
			bufferingPackets = append(bufferingPackets, p)
		buffering:
			for {
				select {
				case <-bufTimer.C:
					break buffering
				case p := <-h.in:
					bufferingPackets = append(bufferingPackets, p)
				}
			}
			err = h.rawWrite(bufferingPackets)
			bufferingPackets = bufferingPackets[:0]
		}
		if err != nil {
			if !strings.Contains(err.Error(), "closed") {
				h.exitError = logex.NewErrorf("write error: %v", err)
			}
			break
		}
	}
}

func (h *HttpChan) readLoop() {
	h.flow.Add(1)
	defer h.flow.DoneAndClose()

	buf := bufio.NewReader(h.conn)
loop:
	for !h.flow.IsClosed() {
		h.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		l2, err := h.ReadL2(buf)
		if err != nil {
			if err, ok := err.(*net.OpError); ok {
				if err.Temporary() || err.Timeout() {
					continue
				}
			}
			if !strings.Contains(err.Error(), "closed") {
				h.exitError = logex.NewErrorf("read error: %v", err)
			}
			break
		}

		if err := l2.Verify(h.session); err != nil {
			h.exitError = logex.NewErrorf("verify error: %v", err)
			break
		}

		if h.IsSvrModeAndUninit() {
			out, err := h.delegate.Init(int(l2.UserId))
			if err != nil {
				h.exitError = logex.NewErrorf("init error: %v", err)
				break
			}
			h.markInit(out)
			h.delegate.OnInited(h)
		}

		ps, err := l2.Unmarshal()
		if err != nil {
			h.exitError = logex.NewErrorf("client error: %v", err)
			break
		}
		for _, p := range ps {
			h.speed.Download(p.Size())
			if !h.onRecePacket(p) {
				break loop
			}
		}
	}
}

func (h *HttpChan) onRecePacket(p *packet.Packet) bool {
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

func (h *HttpChan) Run() {
	go h.writeLoop()
	go h.readLoop()
}

func (h *HttpChan) GetUserId() (int, error) {
	return h.session.UserId(), nil
}

func (h *HttpChan) Close() {
	if !h.flow.MarkExit() {
		return
	}

	if h.exitError != nil {
		logex.Info(h.Name(), "exit by:", h.exitError)
	} else {
		logex.Info(h.Name(), "exit manually")
	}

	h.flow.Close()
	h.conn.Close()
}

func (h *HttpChan) Name() string {
	return fmt.Sprintf("[%v -> %v]",
		h.conn.LocalAddr(),
		h.conn.RemoteAddr(),
	)
}

func (h *HttpChan) GetStat() *statistic.HeartBeat {
	return h.heartBeat.GetStat()
}

func (h *HttpChan) Latency() (time.Duration, time.Duration) {
	return h.heartBeat.GetLatency()
}

func (h *HttpChan) GetSpeed() *statistic.SpeedInfo {
	return h.speed.GetSpeed()
}

func (h *HttpChan) ChanWrite() chan<- *packet.Packet {
	return h.in
}
