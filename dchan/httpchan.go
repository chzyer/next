package dchan

import (
	"time"

	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/statistic"
)

var _ Channel = &HttpChan{}

// to simulate http interactive
type HttpChan struct{}

func NewHttpChan() *HttpChan {
	return nil
}

func (h *HttpChan) AddOnClose(func()) {

}

func (h *HttpChan) Close() {
}

func (h *HttpChan) Name() string {
	return ""
}

func (h *HttpChan) GetStat() *packet.HeartBeatStat {
	return nil
}

func (h *HttpChan) Latency() (time.Duration, time.Duration) {
	return 0, 0
}

func (h *HttpChan) GetSpeed() *statistic.SpeedInfo {
	return nil
}

func (h *HttpChan) ChanWrite() chan<- *packet.Packet {
	return nil
}
