package packet

import (
	"crypto/rand"
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
		panic("resp can't reply")
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
	if IsHasLoopbackPrefix && t == DATA {
		p.Payload = p.Payload[len(loopbackPrefix):]
	}
	return p, nil
}

func (p *Packet) Data() []byte {
	if IsHasLoopbackPrefix && p.Type == DATA {
		b := make([]byte, len(p.Payload)+len(loopbackPrefix))
		copy(b, loopbackPrefix)
		copy(b[4:], p.Payload)
		return b
	}
	return p.Payload
}

type ReqIder interface {
	GetReqId() uint32
}

func (p *Packet) InitIV(reqId ReqIder) *IV {
	if p.IV == nil {
		p.initIV(reqId.GetReqId())
	}
	return p.IV
}

func (p *Packet) initIV(reqId uint32) {
	p.IV = LazyIV(reqId)
}

func (p *Packet) Marshal(s *SessionIV) []byte {
	// 1. iv [0:16]
	// 2. crc32(payload+type) [16:20]
	// 3. aes(crc32(payload+type), token, iv) [20: 24]
	// 4. len(n+rand(n)) + payload + type) [24: 26]
	// 5. aes( (n+rand(n)) + payload + type, token, iv) [26:]
	//     n ~ [32,128]
	if p.IV == nil {
		switch p.Type {
		case AUTH, AUTH_R, HEARTBEAT, HEARTBEAT_R:
			p.initIV(0)
		default:
			panic("iv is null, " + p.Type.String())
		}
	}
	p.IV.Init(s)

	bodyLength := len(p.Payload) + 1
	totalLength := 26 + bodyLength
	// make fixed length
	randLength := 256
	if totalLength > randLength {
		if totalLength < 1500 {
			randLength = 32
		} else {
			randLength = 0
		}
	} else {
		if totalLength < 64 {
			randLength = 64 - totalLength
		} else {
			randLength -= totalLength
		}
	}
	buffer := make([]byte, 2+randLength+totalLength)

	// 5.3, n + rand(n), 26 = header
	randBuffer := buffer[26 : 26+2+randLength]
	binary.BigEndian.PutUint16(randBuffer, uint16(randLength))
	rand.Read(randBuffer[2:])

	// 5.2, fill payload and type
	copy(buffer[len(buffer)-1:], p.Type.Bytes())
	body := buffer[26:]
	copy(body[2+randLength:], p.Payload)

	checksum := crypto.Crc32(body)

	// 5.1, aes-256-cfb
	s.Encode(p.IV, body, body)

	// 4, length, we already make sure bodyLength will not overflowed
	binary.BigEndian.PutUint16(buffer[24:26], uint16(bodyLength+2+randLength))
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
	length := binary.BigEndian.Uint16(header[8:10]) // n + rand(n) + bodyLength

	// at least we have `type`
	if length < 1+2 || length > uint16(MaxPayloadLength) {
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

	randN := binary.BigEndian.Uint16(aesBody[:2])
	payloadOffset := 2 + int(randN)

	var t Type
	payload := aesBody[payloadOffset : len(aesBody)-1]
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
