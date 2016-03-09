package packet

import "fmt"

type Type int

// packet type
const (
	_        = Type(iota)
	Auth     // auth req, ignore
	AuthResp // ignore
	Data     // use to transfer data
	DataResp // ack the transfer

	HeartBeat
	HeartBeatResp

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
	case Auth:
		return "Auth"
	case AuthResp:
		return "AuthResp"
	case Data:
		return "Data"
	case DataResp:
		return "DataResp"
	case HeartBeat:
		return "HeartBeat"
	case HeartBeatResp:
		return "HeartBeatResp"
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
