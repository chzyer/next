package dchan

import (
	"sync"

	"github.com/chzyer/flow"
	"github.com/chzyer/logex"
)

// Server side for operate datachannel for all user
type Server struct {
	flow     *flow.Flow
	group    map[int]*Group // map[userId]Group
	delegate SvrDelegate
	m        sync.RWMutex
}

func NewServer(f *flow.Flow, delegate SvrDelegate) *Server {
	s := &Server{
		delegate: delegate,
		group:    make(map[int]*Group),
	}
	f.ForkTo(&s.flow, s.Close)
	return s
}

func (s *Server) Group(userId int) (*Group, error) {
	fromUser, toUser, err := s.delegate.GetUserChannelFromDataChannel(int(userId))
	if err != nil {
		return nil, err
	}

	s.m.RLock()
	group := s.group[userId]
	s.m.RUnlock()
	if group == nil {
		s.m.Lock()
		group = s.group[userId]
		if group == nil {
			group = NewGroup(s.flow, fromUser, toUser)
			group.Run()
			s.group[userId] = group
		}
		s.m.Unlock()
	}
	return group, nil
}

func (s *Server) AddChannel(ch Channel) error {
	userId, err := ch.GetUserId()
	if err != nil {
		return err
	}
	group, err := s.Group(userId)
	if err != nil {
		return err
	}
	group.AddWithAutoRemove(ch)
	return nil
}

func (s *Server) Close() {
	if !s.flow.MarkExit() {
		return
	}
	s.flow.Close()
	logex.Info("closed")
}
