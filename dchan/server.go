package dchan

import (
	"container/list"
	"sync"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/datachannel"
	"github.com/chzyer/next/util"
)

type Server struct {
	flow           *flow.Flow
	delegate       datachannel.SvrDelegate
	listenerCnt    util.AtomicInt
	running        util.AtomicInt
	listeners      *list.List
	onListenerExit chan struct{}
	mutex          sync.RWMutex
}

// server communicate with channel
func NewServer(f *flow.Flow, delegate datachannel.SvrDelegate) *Server {
	s := &Server{
		delegate:       delegate,
		listeners:      list.New(),
		onListenerExit: make(chan struct{}, 1),
	}
	f.ForkTo(&s.flow, s.Close)
	return s
}

func (s *Server) Close() {
	if !s.flow.MarkExit() {
		return
	}
	s.flow.Close()
}

func (s *Server) loop() {
	s.flow.Add(1)
	defer s.flow.DoneAndClose()

loop:
	for !s.flow.IsClosed() {
		if s.running.Val() < s.listenerCnt.Val() {
			s.running.Add(1)
			s.addNewListener()
			s.delegate.OnDChanUpdate(s.GetAllDataChannel())
		} else {
			select {
			case <-s.onListenerExit:
				s.running.Add(-1)
			case <-s.flow.IsClose():
				break loop
			}
		}
	}
}

func (s *Server) findListener(f func(ln *datachannel.Listener) bool) (ln *list.Element) {
	s.mutex.RLock()
	for elem := s.listeners.Front(); elem != nil; elem = elem.Next() {
		if f(elem.Value.(*datachannel.Listener)) {
			ln = elem
			break
		}
	}
	s.mutex.RUnlock()
	return ln
}

func (s *Server) removeListener(ln *datachannel.Listener) {
	s.mutex.Lock()
	for elem := s.listeners.Front(); elem != nil; elem = elem.Next() {
		if elem.Value.(*datachannel.Listener) == ln {
			s.listeners.Remove(elem)
			break
		}
	}
	s.mutex.Unlock()

	select {
	case s.onListenerExit <- struct{}{}:
	default:
	}

}

func (s *Server) addNewListener() error {
	var ln *datachannel.Listener
	var err error
	ln, err = datachannel.NewListener(s.flow, s.delegate, func() {
		s.removeListener(ln)
	})
	if err != nil {
		return logex.Trace(err)
	}

	s.mutex.Lock()
	s.listeners.PushBack(ln)
	s.mutex.Unlock()

	go ln.Serve()
	return nil
}

func (s *Server) Run(n int) {
	s.listenerCnt.Store(n)
	go s.loop()
}

func (s *Server) GetDataChannel() int {
	ports := s.GetAllDataChannel()
	if len(ports) == 0 {
		return -1
	}
	return util.RandChoiseInt(ports)
}

func (s *Server) GetAllDataChannel() []int {
	s.mutex.RLock()
	ret := make([]int, 0, s.listeners.Len())
	for elem := s.listeners.Front(); elem != nil; elem = elem.Next() {
		ret = append(ret, elem.Value.(*datachannel.Listener).GetPort())
	}
	s.mutex.RUnlock()
	return ret
}
