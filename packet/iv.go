package packet

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/chzyer/next/crypto"
	"gopkg.in/logex.v1"
)

var (
	reqId           = new(uint32)
	ErrPortNotMatch = logex.Define("port %v is not matched")
	ErrUserNotMatch = logex.Define("user %v is not matched")
)

func GetReqId() uint32 {
	return atomic.LoadUint32(reqId)
}

type SessionIV struct {
	UserId uint16
	Token  []byte
	Port   uint16
	Rand   *rand.Rand
}

func NewSessionIV(userId, port uint16, token []byte) *SessionIV {
	if len(token) != 30 { // we only need aes256
		panic("please make sure len(token) == 30")
	}

	// put port number to the last of token
	myToken := make([]byte, 32)
	copy(myToken, token)
	binary.BigEndian.PutUint16(myToken[len(myToken)-2:], port)

	return &SessionIV{
		UserId: userId,
		Port:   port,
		Token:  myToken,
		Rand:   rand.New(rand.NewSource(time.Now().Unix())),
	}
}

func (c *SessionIV) Encode(iv *IV, dst, src []byte) {
	crypto.EncodeAes(dst, src, c.Token, iv.Data)
}

func (c *SessionIV) Decode(iv *IV, dst, src []byte) {
	crypto.DecodeAes(dst, src, c.Token, iv.Data)
}

func (c *SessionIV) Verify(iv *IV) error {
	if iv.Port != c.Port {
		return ErrPortNotMatch.Format(iv.Port)
	}
	if iv.UserId != c.UserId {
		return ErrUserNotMatch.Format(iv.UserId)
	}
	return nil
}

func (c *SessionIV) GenIV() []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 16))
	binary.Write(buf, binary.BigEndian, c.UserId)                   // 2
	binary.Write(buf, binary.BigEndian, c.Port)                     // 2
	binary.Write(buf, binary.BigEndian, atomic.AddUint32(reqId, 1)) // 4
	binary.Write(buf, binary.BigEndian, rand.Int63())               // 8
	return buf.Bytes()
}

type IV struct {
	UserId uint16
	Port   uint16
	ReqId  uint32
	Data   []byte
}

func ParseIV(byte []byte) *IV {
	if len(byte) != 16 {
		panic("please make sure len(byte) = 16")
	}
	buf := bytes.NewReader(byte)
	var iv IV
	iv.Data = byte
	binary.Read(buf, binary.BigEndian, &iv.UserId)
	binary.Read(buf, binary.BigEndian, &iv.Port)
	binary.Read(buf, binary.BigEndian, &iv.ReqId)
	return &iv
}

func ReadIV(r io.Reader) (*IV, error) {
	iv := make([]byte, 16)
	if _, err := io.ReadFull(r, iv); err != nil {
		return nil, logex.Trace(err)
	}
	return ParseIV(iv), nil
}
