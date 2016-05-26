package dchan

import (
	"net"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

type UdpChanFactory struct {
	port int32
}

func NewUdpChanFactory() *UdpChanFactory {
	return &UdpChanFactory{
		port: 10000,
	}
}

func (u *UdpChanFactory) Listen(f *flow.Flow) (net.Listener, error) {
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	return NewUDPListener(f, conn), nil
}

func (UdpChanFactory) DialTimeout(host string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("udp", host, timeout)
}

func (UdpChanFactory) NewClient(f *flow.Flow, session *packet.Session, conn net.Conn, out chan<- *packet.Packet) Channel {
	return NewTcpChanClient(f, session, conn, out)
}

func (UdpChanFactory) NewServer(f *flow.Flow, session *packet.Session, conn net.Conn, delegate SvrInitDelegate) Channel {
	return NewTcpChanServer(f, session, conn, delegate)
}
