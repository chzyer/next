package packet

import (
	"crypto/rand"
	"testing"

	"github.com/chzyer/test"
)

func TestPacket(t *testing.T) {
	defer test.New(t)
	payload := make([]byte, 24)
	rand.Read(payload)
	packet := New(payload, AUTH)
	packet.ReqId = 1

	data := make([]byte, packet.TotalSize())
	packet.Marshal(data)

	packetDst, err := Unmarshal(data)
	test.Nil(err)

	test.Equal(packetDst.ReqId, packet.ReqId)
	test.Equal(packetDst.Type, AUTH)
	test.Equal(packetDst.Payload(), payload)
	test.Equal(packetDst, packet)

}

func BenchmarkPacketUnmarshal(b *testing.B) {
	defer test.New(b)
	payload := make([]byte, 24)
	rand.Read(payload)
	packet := New(payload, DATA)
	packet.ReqId = 1
	data := make([]byte, packet.TotalSize())
	packet.Marshal(data)

	for i := 0; i < b.N; i++ {
		_, err := Unmarshal(data)
		test.Nil(err)
		b.SetBytes(int64(len(data)))
	}
}

func BenchmarkPacketMarshal(b *testing.B) {
	defer test.New(b)
	payload := make([]byte, 24)
	rand.Read(payload)

	for i := 0; i < b.N; i++ {
		packet := New(payload, DATA)
		packet.ReqId = 1
		data := make([]byte, packet.TotalSize())
		packet.Marshal(data)
		b.SetBytes(int64(len(payload)))
	}
}
