package dchan

import (
	"time"

	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/statistic"
)

type Channel interface {
	Close()
	Name() string
	GetStat() *packet.HeartBeatStat
	Latency() (time.Duration, time.Duration)
	AddOnClose(func())
	GetSpeed() *statistic.SpeedInfo
	ChanWrite() chan<- *packet.Packet
}
