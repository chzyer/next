package dchan

import (
	"fmt"
	"net"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/statistic"
)

var (
	_ Channel        = new(HttpChan)
	_ ChannelFactory = new(HttpChanFactory)
)

// to simulate http interactive
type HttpChan struct {
	flow    *flow.Flow
	session *packet.SessionIV
	in      chan *packet.Packet
	fromDC  chan<- *packet.Packet
	onClose func()
	conn    net.Conn

	heartBeat *statistic.HeartBeatStage
	speed     *statistic.Speed
	exitError error
}

func NewHttpChan(f *flow.Flow, session *packet.SessionIV, conn net.Conn, out chan<- *packet.Packet) *HttpChan {
	hc := &HttpChan{
		session: session,
		fromDC:  out,
		conn:    conn,
		speed:   statistic.NewSpeed(),
		in:      make(chan *packet.Packet),
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
	h.onClose = f
}

func (h *HttpChan) Run() {

}

func (h *HttpChan) GetUserId() int {
	return int(h.session.UserId)
}

func (h *HttpChan) Close() {
	if !h.flow.MarkExit() {
		return
	}
	h.flow.Close()
	h.conn.Close()
	if h.onClose != nil {
		h.onClose()
	}
}

func (h *HttpChan) Name() string {
	return ""
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
