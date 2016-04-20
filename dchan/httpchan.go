package dchan

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/statistic"
)

var (
// _ Channel        = new(HttpChan)
// _ ChannelFactory = new(HttpChanFactory)
)

// to simulate http interactive
type HttpChan struct {
	flow    *flow.Flow
	session *packet.Session
	in      chan *packet.Packet
	out     chan<- *packet.Packet
	onClose func()
	conn    net.Conn

	heartBeat *statistic.HeartBeatStage
	speed     *statistic.Speed
	exitError error
}

func NewHttpChanClient(f *flow.Flow, session *packet.Session, conn net.Conn, out chan<- *packet.Packet) *HttpChan {
	hc := &HttpChan{
		session: session,
		out:     out,
		conn:    conn,
		speed:   statistic.NewSpeed(),
		in:      make(chan *packet.Packet),
	}
	f.ForkTo(&hc.flow, hc.Close)
	hc.heartBeat = statistic.NewHeartBeatStage(hc.flow, 5*time.Second, hc)
	return hc
}

func NewHttpChanServer(f *flow.Flow, conn net.Conn, delegate SvrInitDelegate) *HttpChan {
	hc := &HttpChan{
		conn:  conn,
		speed: statistic.NewSpeed(),
		in:    make(chan *packet.Packet),
	}
	f.ForkTo(&hc.flow, hc.Close)
	hc.heartBeat = statistic.NewHeartBeatStage(hc.flow, 5*time.Second, hc)
	return hc
}

func (h *HttpChan) HeartBeatClean(err error) {
	h.exitError = fmt.Errorf("clean: %v", err)
	h.Close()
}

func (h *HttpChan) AddOnClose(f func()) {
	h.flow.AddOnClose(f)
}

func (h *HttpChan) rawWrite(p *packet.Packet) error {
	return nil
}

func (h *HttpChan) writeLoop() {
	h.flow.Add(1)
	defer h.flow.DoneAndClose()

	heartBeatTicker := time.NewTicker(1 * time.Second)
	defer heartBeatTicker.Stop()

	var err error
loop:
	for {
		select {
		case <-h.flow.IsClose():
			break loop
		case <-heartBeatTicker.C:
			p := h.heartBeat.New()
			err = h.rawWrite(p)
			h.heartBeat.Add(p)
		case p := <-h.in:
			err = h.rawWrite(p)
		}
		if err != nil {
			if !strings.Contains(err.Error(), "closed") {
				h.exitError = fmt.Errorf("write error: %v", err)
			}
			break
		}
	}
}

func (h *HttpChan) readPacketFromRequest(r *http.Request) ([]*packet.Packet, error) {
	return nil, nil
}

func (h *HttpChan) readLoop() {
	h.flow.Add(1)
	defer h.flow.DoneAndClose()

	buf := bufio.NewReader(h.conn)
loop:
	for !h.flow.IsClosed() {
		h.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		req, err := http.ReadRequest(buf)
		if err != nil {
			if err, ok := err.(*net.OpError); ok {
				if err.Temporary() || err.Timeout() {
					continue
				}
			}
			if !strings.Contains(err.Error(), "closed") {
				h.exitError = fmt.Errorf("read error: %v", err)
			}
			break
		}
		packets, err := h.readPacketFromRequest(req)
		if err != nil {
			h.exitError = fmt.Errorf("client error: %v", err)
			break
		}
		for _, p := range packets {
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
		logex.DownLevel(1).Debug("where is exit")
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
