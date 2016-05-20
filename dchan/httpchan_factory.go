package dchan

import (
	"net"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

var _ ChannelFactory = new(HttpChanFactory)

type HttpChanFactory struct{}

func (HttpChanFactory) NewClient(f *flow.Flow, s *packet.Session, c net.Conn, o chan<- *packet.Packet) Channel {
	return NewHttpChanClient(f, s, c, o)
}

func (HttpChanFactory) NewServer(f *flow.Flow, s *packet.Session, c net.Conn, d SvrInitDelegate) Channel {
	return NewHttpChanServer(f, s, c, d)
}
