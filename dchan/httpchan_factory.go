package dchan

import (
	"io"
	"net"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

type HttpChanFactory struct{}

func (HttpChanFactory) NewClient(f *flow.Flow, s *packet.Session, c net.Conn, o chan<- *packet.Packet) Channel {
	return NewHttpChanClient(f, s, c, o)
}

func (HttpChanFactory) NewServer(f *flow.Flow, s *packet.Session, c net.Conn, d SvrInitDelegate) Channel {
	return NewHttpChanServer(f, c, d)
}

func (HttpChanFactory) ReadL2(r io.Reader) (*packet.PacketL2, error) {
	return nil, nil
}
