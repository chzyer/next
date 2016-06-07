package dchan

import (
	"net"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	kcp "github.com/xtaci/kcp-go"
)

type UdpChanFactory struct {
	port int32
}

func NewUdpChanFactory() *UdpChanFactory {
	return &UdpChanFactory{
		port: 10000,
	}
}

type wrapLn struct {
	*kcp.Listener
}

func (w *wrapLn) Accept() (net.Conn, error) {
	return w.Listener.Accept()
}

func (u *UdpChanFactory) Listen(f *flow.Flow) (net.Listener, error) {
	ln, err := kcp.Listen(":0")
	if err != nil {
		return nil, err
	}
	return &wrapLn{ln}, nil
}

func (UdpChanFactory) DialTimeout(host string, timeout time.Duration) (net.Conn, error) {
	return kcp.Dial(host)
}

func (UdpChanFactory) NewClient(f *flow.Flow, session *packet.Session, conn net.Conn, out chan<- *packet.Packet) Channel {
	return NewHttpChanClient(f, session, conn, out)
}

func (UdpChanFactory) NewServer(f *flow.Flow, session *packet.Session, conn net.Conn, delegate SvrInitDelegate) Channel {
	return NewHttpChanServer(f, session, conn, delegate)
}
