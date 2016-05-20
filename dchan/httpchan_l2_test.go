package dchan

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/chzyer/logex"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/util"
	"github.com/chzyer/test"
)

func TestHttpChanL2(t *testing.T) {
	defer test.New(t)
	token := util.RandStr(32)
	session := packet.NewSessionCli(0, []byte(token))
	p := packet.New([]byte(util.RandStr(24)), packet.DATA)
	l2 := packet.WrapL2(session, []*packet.Packet{p})

	hc := new(HttpChan)
	buf := bytes.NewBuffer(nil)
	_, err := buf.Write(hc.WriteL2(l2))
	test.Nil(err)

	l22, err := hc.ReadL2(bufio.NewReader(buf))
	test.Nil(err)
	test.Nil(l22.Verify(session))
	ps, err := l22.Unmarshal()
	test.Nil(err)
	logex.Info(ps)
}

func BenchmarkHttpChanL2(b *testing.B) {
	defer test.New(b)
	token := util.RandStr(32)
	session := packet.NewSessionCli(0, []byte(token))
	p := packet.New([]byte(util.RandStr(24)), packet.DATA)
	l2 := packet.WrapL2(session, []*packet.Packet{p})

	hc := new(HttpChan)
	buf := bytes.NewBuffer(nil)
	for i := 0; i < b.N; i++ {
		n, err := buf.Write(hc.WriteL2(l2))
		test.Nil(err)
		b.SetBytes(int64(n))
	}
}

func BenchmarkHttpChanReadL2(b *testing.B) {
	defer test.New(b)
	token := util.RandStr(32)
	session := packet.NewSessionCli(0, []byte(token))
	p := packet.New([]byte(util.RandStr(24)), packet.DATA)
	l2 := packet.WrapL2(session, []*packet.Packet{p})

	hc := new(HttpChan)
	buf := bytes.NewBuffer(nil)

	r := bufio.NewReader(buf)
	for i := 0; i < b.N; i++ {
		buf.Write(hc.WriteL2(l2))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := hc.ReadL2(r)
		test.Nil(err)
		b.SetBytes(24)
	}
}

func BenchmarkTcpChanReadL2(b *testing.B) {
	defer test.New(b)
	token := util.RandStr(32)
	session := packet.NewSessionCli(0, []byte(token))
	p := packet.New([]byte(util.RandStr(24)), packet.DATA)
	l2 := packet.WrapL2(session, []*packet.Packet{p})

	hc := new(TcpChan)
	buf := bytes.NewBuffer(nil)

	r := bufio.NewReader(buf)
	for i := 0; i < b.N; i++ {
		buf.Write(hc.WriteL2(l2))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := hc.ReadL2(r)
		test.Nil(err)
		b.SetBytes(24)
	}
}
