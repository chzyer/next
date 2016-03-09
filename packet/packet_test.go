package packet

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/chzyer/next/test"
)

func TestPacket(t *testing.T) {
	defer test.New(t)
	payload := make([]byte, 24)
	rand.Read(payload)
	packet, err := New(payload, Auth)
	test.Nil(err)

	token := make([]byte, 30)
	rand.Read(payload)
	port := uint16(1024)
	userId := uint16(1)
	s := NewSessionIV(userId, port, token)
	data := packet.Marshal(s)
	packetDst, err := Read(s, bytes.NewReader(data))
	test.Nil(err)

	test.Equal(packetDst.IV.Port, port)
	test.Equal(packetDst.IV.UserId, userId)
	test.Equal(packetDst.Type, Auth)
	test.Equal(packetDst.Payload, payload)
	test.Equal(packetDst, packet)

}
