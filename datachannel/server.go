package datachannel

import (
	"time"

	"github.com/chzyer/flow"
	"gopkg.in/logex.v1"
)

type Server struct {
	flow      *flow.Flow
	delegate  SvrDelegate
	listeners []*Listener
}

func NewServer(f *flow.Flow, d SvrDelegate) *Server {
	m := &Server{
		flow:     f,
		delegate: d,
	}
	return m
}

func (m *Server) GetDataChannel() int {
	return m.listeners[0].GetPort()
}

func (m *Server) Start(n int) {
	m.flow.Add(1)
	defer m.flow.DoneAndClose()

	started := 0
loop:
	for !m.flow.IsClosed() {
		if started < n {
			m.AddChannelListener()
			started++
		} else {
			select {
			case <-m.flow.IsClose():
				break loop
			case <-time.After(time.Second):
			}
		}
	}
}

func (m *Server) AddChannelListener() error {
	ln, err := NewListener(m.flow, m.delegate)
	if err != nil {
		return logex.Trace(err)
	}
	m.listeners = append(m.listeners, ln)

	go ln.Serve()
	return nil
}
