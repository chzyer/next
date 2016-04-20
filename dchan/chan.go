package dchan

import (
	"net"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/statistic"
)

type SvrDelegate interface {
	SvrAuthDelegate
	GetUserChannelFromDataChannel(id int) (
		fromUser <-chan *packet.Packet, toUser chan<- *packet.Packet, err error)
	OnDChanUpdate([]int)
	OnNewChannel(Channel)
}

type SvrInitDelegate interface {
	Init(id int) (toUser chan<- *packet.Packet, err error)
	OnInited(ch Channel)
}

type SvrAuthDelegate interface {
	GetUserToken(id int) ([]byte, error)
}

type ChannelFactory interface {
	NewClient(*flow.Flow, *packet.Session, net.Conn, chan<- *packet.Packet) Channel
	NewServer(*flow.Flow, *packet.Session, net.Conn, SvrInitDelegate) Channel
}

type Channel interface {
	Close()
	Name() string
	GetStat() *statistic.HeartBeat
	Latency() (time.Duration, time.Duration)
	GetUserId() (int, error)
	AddOnClose(func())
	GetSpeed() *statistic.SpeedInfo
	ChanWrite() chan<- *packet.Packet
	Run()
}
