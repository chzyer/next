package packet

import (
	"encoding/binary"
	"fmt"
	"math"
	"runtime"

	"github.com/chzyer/flow"
	"github.com/chzyer/logex"
)

var (
	IsHasLoopbackPrefix = runtime.GOOS == "darwin"
	loopbackPrefix      = []byte{0, 0, 0, 2}
	MaxPayloadLength    = math.MaxUint16 - 27 // header(26) + type(1)
)

var (
	ErrPacketTooShort  = logex.Define("packet too short: %v")
	ErrInvalidType     = logex.Define("invalid type: %v")
	ErrInvalidToken    = logex.Define("invalid token")
	ErrInvalidLength   = logex.Define("invalid length, want:%v, got: %v")
	ErrPayloadTooLarge = logex.Define("payload is too large: %v")
)

type RecvChan <-chan []*Packet

func (c RecvChan) RecvAll(f *flow.Flow) []*Packet {
	select {
	case p := <-c:
		return p
	case <-f.IsClose():
		return nil
	}
}

type SendChan chan<- []*Packet

func (s SendChan) SendSafe(f *flow.Flow, p []*Packet) bool {
	select {
	case s <- p:
		return true
	case <-f.IsClose():
		return false
	}
}

func (s SendChan) SendOneSafe(f *flow.Flow, p *Packet) bool {
	select {
	case s <- []*Packet{p}:
		return true
	case <-f.IsClose():
		return false
	}
}

type Chan chan []*Packet

func NewChan(n int) Chan {
	return make(chan []*Packet, n)
}

func (c Chan) Recv() RecvChan {
	return RecvChan(chan []*Packet(c))
}

func (c Chan) Send() SendChan {
	return SendChan(chan []*Packet(c))
}

func (ch Chan) SendSafe(f *flow.Flow, p []*Packet) bool {
	select {
	case ch <- p:
		return true
	case <-f.IsClose():
		return false
	}
}

// ReqId + Type + Payload
type Packet struct {
	ReqId   uint32
	Type    Type
	payload []byte

	size int
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
	newP, err := newPacket(payload, Type(p.Type+1))
	if err != nil {
		panic(err)
	}
	newP.ReqId = p.ReqId
	return newP
}

func newPacket(payload []byte, t Type) (*Packet, error) {
	if t.IsInvalid() {
		return nil, ErrInvalidType.Format(int(t))
	}
	if len(payload) > MaxPayloadLength {
		return nil, ErrPayloadTooLarge.Format(len(payload))
	}
	if IsHasLoopbackPrefix && t == DATA {
		payload = payload[len(loopbackPrefix):]
	}

	p := &Packet{
		Type:    t,
		payload: payload,
		size:    len(payload),
	}
	return p, nil
}

func (p *Packet) Size() int {
	return p.size
}

func (p *Packet) Payload() []byte {
	if IsHasLoopbackPrefix && p.Type == DATA {
		b := make([]byte, len(p.payload)+len(loopbackPrefix))
		copy(b, loopbackPrefix)
		copy(b[4:], p.payload)
		return b
	}
	return p.payload
}

type Reqider interface {
	GetReqId() uint32
}

func (p *Packet) SetReqId(r Reqider) {
	if p.ReqId == 0 {
		p.ReqId = r.GetReqId()
	}
}

func (p *Packet) Marshal(ret []byte) int {
	// ret := make([]byte, 8+len(p.payload)) // reqId(4) + type(2) + len(payload)
	binary.BigEndian.PutUint32(ret[:4], p.ReqId)
	binary.BigEndian.PutUint16(ret[4:6], uint16(p.Type))
	binary.BigEndian.PutUint16(ret[6:8], uint16(len(p.payload)))
	n := copy(ret[8:], p.payload)
	if n != len(p.payload) {
		panic(fmt.Sprintf("short written: %v, want:%v, bufferSize: %v, totalSize: %v",
			n, len(p.payload), len(ret), p.TotalSize()))
	}
	return n + 8
}

func (p *Packet) TotalSize() int {
	return 8 + p.size
}

func Unmarshal(b []byte) (*Packet, error) {
	if len(b) < 8 {
		return nil, ErrPacketTooShort.Format(len(b))
	}
	reqId := binary.BigEndian.Uint32(b[:4])
	typ := binary.BigEndian.Uint16(b[4:6])
	length := binary.BigEndian.Uint16(b[6:8])
	payload := make([]byte, int(length))
	if len(b[8:]) < int(length) {
		return nil, ErrInvalidLength.Format(int(length), len(b[8:]))
	}
	copy(payload, b[8:])
	return &Packet{
		ReqId:   reqId,
		Type:    Type(typ),
		payload: payload,
		size:    int(length),
	}, nil
}
