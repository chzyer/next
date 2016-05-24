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
	packets := make([]*packet.Packet, 20)
	for idx := range packets {
		packets[idx] = p
	}

	l2 := packet.WrapL2(session, packets)

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
	packets := make([]*packet.Packet, 20)
	for idx := range packets {
		packets[idx] = p
	}

	l2 := packet.WrapL2(session, packets)

	hc := new(HttpChan)
	buf := bytes.NewBuffer(nil)

	r := bufio.NewReader(buf)
	for i := 0; i < b.N; i++ {
		buf.Write(hc.WriteL2(l2))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pl2, err := hc.ReadL2(r)
		test.Nil(err)
		test.Nil(pl2.Verify(session))
		ps, err := pl2.Unmarshal()
		test.Nil(err)
		tt := 0
		for i := 0; i < len(ps); i++ {
			tt += ps[i].Size()
		}
		b.SetBytes(int64(tt))
	}
}

func BenchmarkTcpChanReadL2(b *testing.B) {
	defer test.New(b)
	token := util.RandStr(32)
	session := packet.NewSessionCli(0, []byte(token))
	p := packet.New([]byte(util.RandStr(24)), packet.DATA)
	packets := make([]*packet.Packet, 20)
	for idx := range packets {
		packets[idx] = p
	}
	l2 := packet.WrapL2(session, packets)

	hc := new(TcpChan)
	buf := bytes.NewBuffer(nil)

	r := bufio.NewReader(buf)
	for i := 0; i < b.N; i++ {
		buf.Write(hc.WriteL2(l2))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l2, err := hc.ReadL2(r)
		test.Nil(err)
		test.Nil(l2.Verify(session))
		ps, err := l2.Unmarshal()
		test.Nil(err)
		t := 0
		for _, p := range ps {
			t += p.Size()
		}
		b.SetBytes(int64(t))
	}
}
