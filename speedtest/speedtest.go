package speedtest

import (
	"crypto/rand"
	"io"

	"github.com/chzyer/next/util"
)

type Speedtest struct {
	Total util.Size
	Sent  util.Size
	w     io.Writer
}

func NewSpeedtest(total int64, w io.Writer) *Speedtest {
	st := &Speedtest{
		Total: util.Size(total),
		w:     w,
	}
	return st
}

func (s *Speedtest) Run() {
	a := make([]byte, 4096)
	var n int
	var err error
	for {
		rand.Read(a)
		n, err = s.w.Write(a)
		if err != nil {

		}
		s.Sent += util.Size(n)
	}
}
