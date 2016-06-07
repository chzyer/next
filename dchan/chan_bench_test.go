package dchan

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/logex"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/test"
)

var token = test.RandBytes(32)
var dataPacket = packet.New(test.RandBytes(128), packet.DATA)

func init() {
	logex.DebugLevel = 2
}

func BenchmarkAllGroup(b *testing.B) {
	defer test.New(b)

	f := flow.New()
	toDC := packet.NewChan(0)
	fromDC := packet.NewChan(0)
	g := NewGroup(f, toDC.Recv(), fromDC.Send())
	go g.Run()
	defer f.Close()

	cf := &HttpChanFactory{}
	ln, err := cf.Listen(f)
	test.Nil(err)
	go testFactoryListen(f, b, cf, ln)

	conn, err := cf.DialTimeout(getAddr(ln.Addr()), time.Second)
	test.Nil(err)

	session := packet.NewSessionCli(0, token)
	ch := cf.NewClient(f, session, conn, fromDC.Send())
	go ch.Run()
	g.AddWithAutoRemove(ch)

	data := make([]*packet.Packet, 10)
	for idx := range data {
		data[idx] = dataPacket
	}

	for i := 0; i < b.N; i++ {
		g.Send(data)
	}
}

func BenchmarkAllHttpChan(b *testing.B) {
	defer test.New(b)
	f := flow.New()
	testFactory(f, b, &HttpChanFactory{})
}

func BenchmarkAllTCPChan(b *testing.B) {
	defer test.New(b)
	f := flow.New()
	testFactory(f, b, &HttpChanFactory{})
}

type dumpSvrInitDelegate struct {
	ch packet.SendChan
}

func (d *dumpSvrInitDelegate) Init(uid int) (packet.SendChan, error) {
	return d.ch, nil
}

func (d *dumpSvrInitDelegate) OnInited(ch Channel) {

}

func getAddr(addr net.Addr) string {
	var port int
	switch addr := addr.(type) {
	case *net.UDPAddr:
		port = addr.Port
	case *net.TCPAddr:
		port = addr.Port
	}

	return fmt.Sprintf("localhost:%v", port)
}

func testFactory(f *flow.Flow, b *testing.B, cf ChannelFactory) {
	ln, err := cf.Listen(f)
	test.Nil(err)
	go testFactoryListen(f, b, cf, ln)
	defer f.Close()

	conn, err := cf.DialTimeout(getAddr(ln.Addr()), time.Second)
	test.Nil(err)
	session := packet.NewSessionCli(0, token)
	ch := packet.NewChan(0)
	cli := cf.NewClient(f, session, conn, ch.Send())
	go cli.Run()
	for i := 0; i < b.N; i++ {
		cli.ChanWrite().SendOneSafe(f, dataPacket)
	}
}

func testFactoryListen(f *flow.Flow, b *testing.B, cf ChannelFactory, ln net.Listener) {
	conn, err := ln.Accept()
	test.Nil(err)

	session := packet.NewSessionCli(0, token)
	ch := packet.NewChan(2)
	delegate := &dumpSvrInitDelegate{ch.Send()}
	svr := cf.NewServer(f, session, conn, delegate)
	go svr.Run()

loop:
	for {
		select {
		case ps := <-ch:
			s := int64(0)
			for _, p := range ps {
				s += int64(p.Size())
			}
			b.SetBytes(s)
		case <-f.IsClose():
			break loop
		}
	}
}
