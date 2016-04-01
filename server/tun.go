package server

import (
	"github.com/chzyer/flow"
	"github.com/chzyer/tunnel"
	"gopkg.in/logex.v1"
)

type Tun struct {
	flow    *flow.Flow
	tun     *tunnel.Instance
	in, out chan []byte
}

func newTun(f *flow.Flow, cfg *Config) (*Tun, error) {
	tun, err := tunnel.New(&tunnel.Config{
		DevId:   cfg.DevId,
		Gateway: cfg.Net.IP.IP(),
		Mask:    cfg.Net.Mask,
		MTU:     cfg.MTU,
		Debug:   cfg.DebugTun,
	})
	if err != nil {
		return nil, logex.Trace(err)
	}
	t := &Tun{
		tun: tun,
		in:  make(chan []byte),
		out: make(chan []byte),
	}
	f.ForkTo(&t.flow, t.Close)
	return t, nil
}

func (t *Tun) WriteChan() chan<- []byte {
	return t.in
}

func (t *Tun) ReadChan() <-chan []byte {
	return t.out
}

func (t *Tun) Run() {
	go t.writeLoop(t.in)
	go t.readLoop(t.out)
}

func (t *Tun) writeLoop(in chan []byte) {
	t.flow.Add(1)
	defer t.flow.DoneAndClose()
loop:
	for {
		select {
		case data := <-in:
			_, err := t.tun.Write(data)
			if err != nil {
				break loop
			}
		case <-t.flow.IsClose():
			break loop
		}
	}
}

func (t *Tun) readLoop(out chan []byte) {
	t.flow.Add(1)
	defer t.flow.DoneAndClose()

	buf := make([]byte, 65536)

loop:
	for {
		n, err := t.tun.Read(buf)
		if err != nil {
			break
		}
		b := make([]byte, n)
		copy(b, buf[:n])
		select {
		case out <- b:
		case <-t.flow.IsClose():
			break loop
		}
	}
}

func (t *Tun) Close() {
	t.tun.Close()
	t.flow.Close()
}
