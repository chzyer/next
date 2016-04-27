package dchan

import (
	"container/list"
	"sync"

	"github.com/chzyer/flow"
	"github.com/chzyer/logex"
	"github.com/chzyer/next/util"
)

type ListenerGroup struct {
	flow           *flow.Flow
	delegate       SvrDelegate
	listenerCnt    util.AtomicInt
	running        util.AtomicInt
	listeners      *list.List
	onListenerExit chan struct{}
	mutex          sync.RWMutex
	chanType       string
}

// server communicate with channel
func NewListenerGroup(f *flow.Flow, chanType string, delegate SvrDelegate) *ListenerGroup {
	s := &ListenerGroup{
		delegate:       delegate,
		listeners:      list.New(),
		onListenerExit: make(chan struct{}, 1),
		chanType:       chanType,
	}
	f.ForkTo(&s.flow, s.Close)
	return s
}

func (s *ListenerGroup) Close() {
	if !s.flow.MarkExit() {
		return
	}
	s.flow.Close()
}

func (s *ListenerGroup) loop() {
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

func (s *ListenerGroup) findListener(f func(ln *Listener) bool) (ln *list.Element) {
	s.mutex.RLock()
	for elem := s.listeners.Front(); elem != nil; elem = elem.Next() {
		if f(elem.Value.(*Listener)) {
			ln = elem
			break
		}
	}
	s.mutex.RUnlock()
	return ln
}

func (s *ListenerGroup) removeListener(ln *Listener) {
	s.mutex.Lock()
	for elem := s.listeners.Front(); elem != nil; elem = elem.Next() {
		if elem.Value.(*Listener) == ln {
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

func (s *ListenerGroup) addNewListener() error {
	var ln *Listener
	var err error
	ln, err = NewListener(s.flow, s.delegate, s.chanType, func() {
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

func (s *ListenerGroup) Run(n int) {
	s.listenerCnt.Store(n)
	go s.loop()
}

func (s *ListenerGroup) GetDataChannel() int {
	ports := s.GetAllDataChannel()
	if len(ports) == 0 {
		return -1
	}
	return util.RandChoiseInt(ports)
}

func (s *ListenerGroup) GetAllDataChannel() []int {
	s.mutex.RLock()
	ret := make([]int, 0, s.listeners.Len())
	for elem := s.listeners.Front(); elem != nil; elem = elem.Next() {
		ret = append(ret, elem.Value.(*Listener).GetPort())
	}
	s.mutex.RUnlock()
	return ret
}
