package dchan

import (
	"net"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/logex"
	"github.com/chzyer/next/packet"
)

var (
	ErrInvalidUserId        = logex.Define("invalid user id")
	ErrUnexpectedPacketType = logex.Define("unexpected packet type")
)

var _ ChannelFactory = TcpChanFactory{}

type TcpChanFactory struct{}

func (TcpChanFactory) Listen(*flow.Flow) (net.Listener, error) {
	return net.Listen("tcp", ":0")
}

func (TcpChanFactory) DialTimeout(host string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", host, timeout)
}

func (TcpChanFactory) NewClient(f *flow.Flow, session *packet.Session, conn net.Conn, out chan<- *packet.Packet) Channel {
	return NewTcpChanClient(f, session, conn, out)
}
func (TcpChanFactory) NewServer(f *flow.Flow, session *packet.Session, conn net.Conn, delegate SvrInitDelegate) Channel {
	return NewTcpChanServer(f, session, conn, delegate)
}
