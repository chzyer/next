package packet

import (
	"crypto/rand"
	"encoding/binary"
	"io"

	"gopkg.in/logex.v1"

	"github.com/chzyer/next/crypto"
	"github.com/chzyer/next/util"
)

const PacketL2HeaderSize = 24

// to verify auth
// iv + userid +          // header (18)
// crc32(payload)         // checksum (4)
// len(payload) + payload // payload (2+n)
type PacketL2 struct {
	IV      []byte
	UserId  uint16
	Payload []byte

	checksum uint32
	verifyd  *error
}

func ReadL2(r io.Reader) (*PacketL2, error) {
	header, err := util.ReadFull(r, PacketL2HeaderSize)
	if err != nil {
		return nil, logex.Trace(err, "read l2 header")
	}

	iv := header[:16]
	userId := binary.BigEndian.Uint16(header[16:18])
	checksum := binary.BigEndian.Uint32(header[18:22])
	length := binary.BigEndian.Uint16(header[22:24])
	payload, err := util.ReadFull(r, int(length))
	if err != nil {
		return nil, logex.Trace(err, "read l2 payload")
	}

	return &PacketL2{
		IV:       iv,
		UserId:   userId,
		Payload:  payload,
		checksum: checksum,
	}, nil
}

func WrapL2(s *Session, p *Packet) *PacketL2 {
	l2 := &PacketL2{
		IV:      make([]byte, 16),
		UserId:  uint16(s.UserId()),
		Payload: p.Marshal(),
	}
	rand.Read(l2.IV)
	l2.checksum = crypto.Crc32(l2.Payload)
	s.Encode(l2.IV, l2.Payload, l2.Payload)
	return l2
}

func (p *PacketL2) Verify(s *Session) error {
	if p.verifyd != nil {
		return logex.Trace(*p.verifyd)
	}

	err := s.Verify(int(p.UserId), p.checksum, p.IV, p.Payload)
	p.verifyd = &err
	return logex.Trace(err)
}

func (p *PacketL2) Marshal() []byte {
	ret := make([]byte, PacketL2HeaderSize+len(p.Payload))
	copy(ret[:16], p.IV)
	binary.BigEndian.PutUint16(ret[16:18], p.UserId)
	binary.BigEndian.PutUint32(ret[18:22], p.checksum)
	binary.BigEndian.PutUint16(ret[22:24], uint16(len(p.Payload)))
	copy(ret[PacketL2HeaderSize:], p.Payload)
	return ret
}
