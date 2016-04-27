package dchan

import (
	"encoding/binary"
	"io"

	"github.com/chzyer/logex"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/util"
)

func (c *TcpChan) ReadL2(r io.Reader) (*packet.PacketL2, error) {
	header, err := util.ReadFull(r, 24)
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

	return packet.NewPacketL2(iv, userId, payload, checksum), nil
}

func (c *TcpChan) MarshalL2(p *packet.PacketL2) []byte {
	ret := make([]byte, 24+len(p.Payload))
	copy(ret[:16], p.IV)
	binary.BigEndian.PutUint16(ret[16:18], p.UserId)
	binary.BigEndian.PutUint32(ret[18:22], p.Checksum)
	binary.BigEndian.PutUint16(ret[22:24], uint16(len(p.Payload)))
	copy(ret[24:], p.Payload)
	return ret
}
