package packet

import "fmt"

type Type int

// packet type
const (
	_ Type = iota

	AUTH   // payload: token
	AUTH_R // payload: token
	DATA   // payload: ip packet
	DATA_R // payload: nil

	HEARTBEAT   // payload: nil
	HEARTBEAT_R // payload: nil

	NEWDC   // payload: json([port])
	NEWDC_R // payload: nil

	// send bytes to remote
	SPEED   // payload: [4096]bytes in random
	SPEED_R // payload: nil

	// let remote send N bytes to local
	SPEED_REQ // payload: byte size(uint64)
	SPEED_REQ_R

	InvalidType
)

func (t Type) IsReq() bool {
	return byte(t)%2 == 1
}

func (t Type) IsResp() bool {
	return byte(t)%2 == 0
}

func (t Type) String() string {
	switch t {
	case AUTH:
		return "Auth"
	case AUTH_R:
		return "AuthResp"
	case DATA:
		return "Data"
	case DATA_R:
		return "DataResp"
	case HEARTBEAT:
		return "HeartBeat"
	case HEARTBEAT_R:
		return "HeartBeatResp"
	case NEWDC:
		return "NewDC"
	case NEWDC_R:
		return "NewDCResp"
	default:
		return fmt.Sprintf("<unknown type>:%v", int(t))
	}
}

func (t Type) IsInvalid() bool {
	return t >= InvalidType || t == 0
}

func (t *Type) Marshal(b []byte) error {
	if len(b) != 1 {
		return ErrInvalidType.Trace()
	}
	*t = Type(b[0])
	if t.IsInvalid() {
		return ErrInvalidToken.Trace()
	}
	return nil
}

func (t Type) Bytes() []byte {
	ret := make([]byte, 1)
	ret[0] = byte(t)
	return ret
}
