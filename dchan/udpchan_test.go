package dchan

import (
	"net"
	"testing"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/test"
)

var data = test.RandBytes(1024)

func handleListener(ln *UDPListener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			break
		}
		test.ReadString(conn, "hello")
		test.WriteString(conn, "hello!too!")
		test.ReadString(conn, "aaa")
		test.Write(conn, data)
	}
}

func TestUdpChan(t *testing.T) {
	defer test.New(t)

	addr, err := net.ResolveUDPAddr("udp", ":0")
	test.Nil(err)
	conn, err := net.ListenUDP("udp", addr)
	test.Nil(err)

	f := flow.New()

	ln := NewUDPListener(f, conn)
	defer ln.Close()
	go handleListener(ln)

	{
		addr := conn.LocalAddr().String()
		conn, err := net.DialTimeout("udp", addr, time.Second)
		test.Nil(err)
		test.WriteString(conn, "hello")
		test.WriteString(conn, "aaa")
		test.ReadString(conn, "hello!too!")
		test.Read(conn, data)
	}
}
