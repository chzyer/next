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

	data := packet.Marshal()

	packetDst, err := Unmarshal(data)
	test.Nil(err)

	test.Equal(packetDst.ReqId, packet.ReqId)
	test.Equal(packetDst.Type, AUTH)
	test.Equal(packetDst.Payload(), payload)
	test.Equal(packetDst, packet)

}
