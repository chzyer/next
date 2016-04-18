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
	OnDChanUpdate([]int)
	OnNewChannel(Channel)
	GetUserChannelFromDataChannel(id int) (
		fromUser <-chan *packet.Packet, toUser chan<- *packet.Packet, err error)
}

type SvrAuthDelegate interface {
	GetUserToken(id int) string
}

type ChannelFactory interface {
	New(*flow.Flow, *packet.SessionIV, net.Conn, chan<- *packet.Packet) Channel
	CliAuth(conn net.Conn, session *packet.SessionIV) error
	SvrAuth(delegate SvrAuthDelegate, conn net.Conn, port int) (*packet.SessionIV, error)
}

type Channel interface {
	Close()
	Name() string
	GetStat() *packet.HeartBeatStat
	Latency() (time.Duration, time.Duration)
	GetUserId() int
	AddOnClose(func())
	GetSpeed() *statistic.SpeedInfo
	ChanWrite() chan<- *packet.Packet
	Run()
}
