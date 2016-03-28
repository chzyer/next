package packet

import "github.com/chzyer/next/ip"

type DataPacket struct {
	*Packet
}

func NewDataPacket(payload []byte) *DataPacket {
	return &DataPacket{New(payload, DATA)}
}

func (d *DataPacket) SrcIP() ip.IP {
	return ip.NewIP(d.Payload[12:16])
}

func (d *DataPacket) DestIP() ip.IP {
	return ip.NewIP(d.Payload[16:20])
}
