package dchan

import (
	"net"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

var _ ChannelFactory = new(HttpChanFactory)

type HttpChanFactory struct{}

func (HttpChanFactory) Listen(*flow.Flow) (net.Listener, error) {
	return net.Listen("tcp", ":0")
}

func (HttpChanFactory) DialTimeout(host string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", host, timeout)
}

func (HttpChanFactory) NewClient(f *flow.Flow, s *packet.Session, c net.Conn, o chan<- *packet.Packet) Channel {
	return NewHttpChanClient(f, s, c, o)
}

func (HttpChanFactory) NewServer(f *flow.Flow, s *packet.Session, c net.Conn, d SvrInitDelegate) Channel {
	return NewHttpChanServer(f, s, c, d)
}
