package controller

import (
	"container/list"
	"sync"
	"time"

	"github.com/chzyer/next/packet"
)

type Stage struct {
	staging map[uint32]*StageRequest
	queue   *list.List
	m       sync.Mutex
}

type StageRequest struct {
	Req  *Request
	Time time.Time
	Elem *list.Element
}

func newStage() *Stage {
	s := &Stage{
		staging: make(map[uint32]*StageRequest),
		queue:   list.New(),
	}
	return s
}

func (s *Stage) Add(p *Request) {
	req := &StageRequest{
		Req:  p,
		Time: time.Now(),
	}
	s.m.Lock()
	req.Elem = s.queue.PushBack(req)
	s.staging[p.Packet.ReqId] = req
	s.m.Unlock()
}

func (s *Stage) Pop(timeout time.Duration) *Request {
	s.m.Lock()
	elem := s.queue.Front()
	if elem != nil {
		sreq := elem.Value.(*StageRequest)
		if time.Now().Sub(sreq.Time) > timeout {
			s.m.Unlock()
			return s.removeLocked(sreq.Req.Packet.ReqId)
		}
	}
	s.m.Unlock()
	return nil
}

func (s *Stage) removeLocked(reqId uint32) (req *Request) {
	sreq := s.staging[reqId]
	if sreq != nil {
		delete(s.staging, reqId)
		s.queue.Remove(sreq.Elem)
		return sreq.Req
	}
	return nil
}

func (s *Stage) Remove(reqId uint32) (req *Request) {
	s.m.Lock()
	req = s.removeLocked(reqId)
	s.m.Unlock()
	return req
}

type StageInfo struct {
	ReqId    uint32
	DataType packet.Type
}

func (s *Stage) ShowStage() []StageInfo {
	s.m.Lock()
	defer s.m.Unlock()
	ret := make([]StageInfo, 0, len(s.staging))
	for k, r := range s.staging {
		ret = append(ret, StageInfo{k, r.Req.Packet.Type})
	}
	return ret
}
