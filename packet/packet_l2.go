package packet

import (
	"crypto/rand"

	"github.com/chzyer/logex"
	"github.com/chzyer/next/crypto"
)

const PacketL2HeaderSize = 24

// to verify auth
// iv + userid +          // header (18)
// crc32(payload)         // checksum (4)
// len(payload) + payload // payload (2+n)
type PacketL2 struct {
	IV       []byte
	UserId   uint16
	Payload  []byte
	Checksum uint32

	verifyd *error
}

func NewPacketL2(iv []byte, userId uint16, payload []byte, checksum uint32) *PacketL2 {
	return &PacketL2{
		IV:       iv,
		UserId:   userId,
		Payload:  payload,
		Checksum: checksum,
	}
}

func WrapL2(s *Session, p *Packet) *PacketL2 {
	l2 := &PacketL2{
		IV:      make([]byte, 16),
		UserId:  uint16(s.UserId()),
		Payload: p.Marshal(),
	}
	rand.Read(l2.IV)
	l2.Checksum = crypto.Crc32(l2.Payload)
	s.Encode(l2.IV, l2.Payload, l2.Payload)
	return l2
}

func (p *PacketL2) Verify(s *Session) error {
	if p.verifyd != nil {
		return logex.Trace(*p.verifyd)
	}

	err := s.Verify(int(p.UserId), p.Checksum, p.IV, p.Payload)
	p.verifyd = &err
	return logex.Trace(err)
}
