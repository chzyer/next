package packet

import (
	"bytes"
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

	token := make([]byte, 32)
	rand.Read(payload)
	userId := uint16(1)
	s := NewSessionCli(int(userId), token)
	l2 := WrapL2(s, packet)
	data := l2.Marshal()

	l2ret, err := ReadL2(bytes.NewReader(data))
	err = l2ret.Verify(s)
	test.Nil(err)
	packetDst, err := Unmarshal(l2ret.Payload)
	test.Nil(err)

	test.Equal(l2ret.UserId, userId)
	test.Equal(packetDst.ReqId, packet.ReqId)
	test.Equal(packetDst.Type, AUTH)
	test.Equal(packetDst.Payload(), payload)
	test.Equal(packetDst, packet)

}
