package dchan

import (
	"net"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

type HttpChanFactory struct{}

func (HttpChanFactory) New(f *flow.Flow, s *packet.SessionIV, c net.Conn, o chan<- *packet.Packet) Channel {
	return NewHttpChan(f, s, c, o)
}

func (HttpChanFactory) CliAuth(conn net.Conn, session *packet.SessionIV) error {
	return nil
}

func (HttpChanFactory) SvrAuth(SvrAuthDelegate, net.Conn, int) (*packet.SessionIV, error) {
	return nil, nil
}
