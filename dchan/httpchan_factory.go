package dchan

import (
	"net"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

type HttpChanFactory struct{}

func (HttpChanFactory) NewClient(f *flow.Flow, s *packet.Session, c net.Conn, o chan<- *packet.Packet) Channel {
	return NewHttpChanClient(f, s, c, o)
}
func (HttpChanFactory) NewServer(f *flow.Flow, c net.Conn, d SvrInitDelegate) Channel {
	return NewHttpChanServer(f, c, d)
}

func (HttpChanFactory) CliAuth(conn net.Conn, session *packet.Session) error {
	return nil
}

func (HttpChanFactory) SvrAuth(SvrAuthDelegate, net.Conn, int) (*packet.Session, error) {
	return nil, nil
}
