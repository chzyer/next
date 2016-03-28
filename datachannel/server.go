package datachannel

import (
	"sync"

	"github.com/chzyer/flow"
	"gopkg.in/logex.v1"
)

type Server struct {
	flow           *flow.Flow
	delegate       SvrDelegate
	listeners      []*Listener
	mutex          sync.Mutex
	onListenerExit chan struct{}
}

func NewServer(f *flow.Flow, d SvrDelegate) *Server {
	m := &Server{
		flow:           f,
		delegate:       d,
		onListenerExit: make(chan struct{}, 1),
	}
	return m
}

func (m *Server) GetDataChannel() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.listeners) == 0 {
		return -1
	}
	return m.listeners[0].GetPort()
}

func (m *Server) GetAllDataChannel() []int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	ret := make([]int, len(m.listeners))
	for idx, ln := range m.listeners {
		ret[idx] = ln.GetPort()
	}
	return ret
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
			case <-m.onListenerExit:
			}
		}
	}
}

func (m *Server) removeListener(idx int) {
	m.mutex.Lock()
	m.listeners = append(m.listeners[:idx], m.listeners[idx+1:]...)
	m.mutex.Unlock()
	select {
	case m.onListenerExit <- struct{}{}:
	default:
	}
}

func (m *Server) AddChannelListener() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	idx := len(m.listeners)
	ln, err := NewListener(m.flow, m.delegate, func() {
		m.removeListener(idx)
	})
	if err != nil {
		return logex.Trace(err)
	}

	m.listeners = append(m.listeners, ln)

	go ln.Serve()
	return nil
}
