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
	toDC := make(chan *packet.Packet)
	fromDC := make(chan *packet.Packet)
	g := NewGroup(f, toDC, fromDC)
	go g.Run()
	defer f.Close()

	cf := &HttpChanFactory{}
	ln, err := cf.Listen(f)
	test.Nil(err)
	go testFactoryListen(f, b, cf, ln)

	conn, err := cf.DialTimeout(getAddr(ln.Addr()), time.Second)
	test.Nil(err)

	session := packet.NewSessionCli(0, token)
	ch := cf.NewClient(f, session, conn, fromDC)
	go ch.Run()
	g.AddWithAutoRemove(ch)

	for i := 0; i < b.N; i++ {
		toDC <- dataPacket
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
	ch chan *packet.Packet
}

func (d *dumpSvrInitDelegate) Init(uid int) (chan<- *packet.Packet, error) {
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
	ch := make(chan *packet.Packet)
	cli := cf.NewClient(f, session, conn, ch)
	go cli.Run()
	for i := 0; i < b.N; i++ {
		cli.ChanWrite() <- dataPacket
	}
}

func testFactoryListen(f *flow.Flow, b *testing.B, cf ChannelFactory, ln net.Listener) {
	conn, err := ln.Accept()
	test.Nil(err)

	session := packet.NewSessionCli(0, token)
	ch := make(chan *packet.Packet, 2)
	delegate := &dumpSvrInitDelegate{ch}
	svr := cf.NewServer(f, session, conn, delegate)
	go svr.Run()

loop:
	for {
		select {
		case p := <-ch:
			b.SetBytes(int64(p.Size()))
		case <-f.IsClose():
			break loop
		}
	}
}
