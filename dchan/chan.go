package dchan

import (
	"bufio"
	"net"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/statistic"
)

type SvrDelegate interface {
	SvrAuthDelegate
	GetUserChannelFromDataChannel(id int) (
		fromUser packet.RecvChan, toUser packet.SendChan, err error)
	OnDChanUpdate([]int)
	OnNewChannel(Channel)
}

type SvrInitDelegate interface {
	Init(id int) (toUser packet.SendChan, err error)
	OnInited(ch Channel)
}

type SvrAuthDelegate interface {
	GetUserToken(id int) ([]byte, error)
}

type ChannelFactory interface {
	NewClient(*flow.Flow, *packet.Session, net.Conn, packet.SendChan) Channel
	NewServer(*flow.Flow, *packet.Session, net.Conn, SvrInitDelegate) Channel

	Listen(f *flow.Flow) (net.Listener, error)
	DialTimeout(host string, timeout time.Duration) (net.Conn, error)
}

type Channel interface {
	Close()
	Name() string
	GetStat() *statistic.HeartBeat
	Latency() (time.Duration, time.Duration)
	GetUserId() (int, error)
	AddOnClose(func())
	GetSpeed() *statistic.SpeedInfo
	ChanWrite() packet.SendChan
	Run()

	ReadL2(*bufio.Reader) (*packet.PacketL2, error)
	WriteL2(*packet.PacketL2) []byte
}
