package packet

import (
	"encoding/binary"
	"io"
	"math"
	"runtime"

	"github.com/chzyer/next/crypto"

	"gopkg.in/logex.v1"
)

var (
	IsHasLoopbackPrefix = runtime.GOOS == "darwin"
	loopbackPrefix      = []byte{0, 0, 0, 2}
	MaxPayloadLength    = math.MaxUint16 - 27 // header(26) + type(1)
)

var (
	ErrInvalidType     = logex.Define("invalid type: %v")
	ErrInvalidToken    = logex.Define("invalid token")
	ErrInvalidLength   = logex.Define("invalid length")
	ErrPayloadTooLarge = logex.Define("payload is too large: %v")
)

type Packet struct {
	IV      *IV // can nil
	Type    Type
	Payload []byte
}

func New(payload []byte, t Type) *Packet {
	p, err := newPacket(payload, t)
	if err != nil {
		panic(err)
	}
	return p
}

func (p *Packet) Reply(payload []byte) *Packet {
	if !p.Type.IsReq() {
		return nil
	}
	return &Packet{
		IV:      p.IV,
		Type:    Type(p.Type + 1),
		Payload: payload,
	}
}

func newPacket(payload []byte, t Type) (*Packet, error) {
	if t.IsInvalid() {
		return nil, ErrInvalidType.Format(int(t))
	}
	if len(payload) > MaxPayloadLength {
		return nil, ErrPayloadTooLarge.Format(len(payload))
	}
	p := &Packet{
		Type:    t,
		Payload: payload,
	}
	if IsHasLoopbackPrefix && t == Data {
		p.Payload = p.Payload[len(loopbackPrefix):]
	}
	return p, nil
}

func (p *Packet) Data() []byte {
	if IsHasLoopbackPrefix && p.Type == Data {
		b := make([]byte, len(p.Payload)+len(loopbackPrefix))
		copy(b, loopbackPrefix)
		copy(b[4:], p.Payload)
		return b
	}
	return p.Payload
}

func (p *Packet) Marshal(s *SessionIV) []byte {
	// 1. iv [0:16]
	// 2. crc32(payload+type) [16:20]
	// 3. aes(crc32(payload+type), token, iv) [20: 24]
	// 4. len(payload+type) [24: 26]
	// 5. aes(payload+type, token, iv) [26:]

	bodyLength := len(p.Payload) + 1
	totalLength := 26 + bodyLength
	buffer := make([]byte, totalLength)
	if p.IV == nil {
		p.IV = ParseIV(s.GenIV())
	}

	// 5.2, fill payload and type
	copy(buffer[len(buffer)-1:], p.Type.Bytes())
	body := buffer[len(buffer)-1-len(p.Payload):]
	copy(body, p.Payload)
	checksum := crypto.Crc32(body)
	// 5.1, aes-256-cfb
	s.Encode(p.IV, body, body)
	// 4, length, we already make sure bodyLength will not overflowed
	binary.BigEndian.PutUint16(buffer[24:26], uint16(bodyLength))
	// 2-3, checksum, aes(checksum)
	binary.BigEndian.PutUint32(buffer[16:20], checksum)
	s.Encode(p.IV, buffer[20:24], buffer[16:20])
	// 1, iv
	copy(buffer[:16], p.IV.Data)
	return buffer
}

func Read(s *SessionIV, r io.Reader) (*Packet, error) {
	// 1. iv [0:16]
	// 2. crc32(payload+type) [16:20]
	// 3. aes(crc32(payload+type), token, iv) [20: 24]
	// 4. len(payload+type) [24: 26]
	// 5. aes(payload+type, token, iv) [26:]
	iv, err := ReadIV(r)
	if err != nil {
		return nil, logex.Trace(err)
	}
	return ReadWithIV(s, iv, r)
}

func ReadWithIV(s *SessionIV, iv *IV, r io.Reader) (*Packet, error) {
	header := make([]byte, 8+2) // s2 + s3 + s4
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, logex.Trace(err)
	}

	checksum := binary.BigEndian.Uint32(header[0:4])
	aesCS := header[4:8]
	length := binary.BigEndian.Uint16(header[8:10])

	// at least we have `type`
	if length < 1 || length > uint16(MaxPayloadLength) {
		return nil, ErrInvalidLength.Trace(length)
	}

	// 2-3, fast fall
	s.Decode(iv, aesCS, aesCS)
	if binary.BigEndian.Uint32(aesCS) != checksum {
		return nil, ErrInvalidToken.Trace()
	}

	// 5
	aesBody := make([]byte, int(length))
	if _, err := io.ReadFull(r, aesBody); err != nil {
		return nil, logex.Trace(err)
	}

	// 5, checksum with body
	s.Decode(iv, aesBody, aesBody)
	if crypto.Crc32(aesBody) != checksum {
		return nil, ErrInvalidToken.Trace("crc:", crypto.Crc32(aesBody), checksum)
	}

	var t Type
	payload := aesBody[:len(aesBody)-1]
	if err := t.Marshal(aesBody[len(aesBody)-1:]); err != nil {
		return nil, logex.Trace(err)
	}

	p := &Packet{
		IV:      iv,
		Payload: payload,
		Type:    t,
	}

	return p, nil
}
