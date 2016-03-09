package clock

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

type Clock struct {
	offset time.Duration
}

func NewByRemote(ts int64) *Clock {
	now := time.Now().Unix()
	return &Clock{offset: time.Duration(ts-now) * time.Second}
}

func New() *Clock {
	data := make([]byte, 2)
	rand.Read(data)
	offset := binary.BigEndian.Uint16(data)
	return &Clock{offset: time.Duration(offset) * time.Second}
}

func (c *Clock) Now() time.Time {
	return time.Now().Add(time.Duration(c.offset))
}

func (c *Clock) Unix() int64 {
	return c.Now().Unix()
}
