package controller

import (
	"sync"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

type Stage struct {
	flow    *flow.Flow
	staging map[uint32]**Request
	queue   []**Request
	m       sync.Mutex
}

func newStage(f *flow.Flow) *Stage {
	s := &Stage{
		staging: make(map[uint32]**Request),
	}
	// f.ForkTo(s.flow, )
	return s
}

type StageInfo struct {
	ReqId    uint32
	DataType packet.Type
}

func (s *Stage) Add(p *Request) {
	s.m.Lock()
	s.staging[p.Packet.IV.ReqId] = &p
	s.m.Unlock()
}

func (s *Stage) Remove(reqId uint32) (req *Request) {
	s.m.Lock()
	reqPtr := s.staging[reqId]
	if reqPtr != nil {
		request := **reqPtr
		req = &request
		*reqPtr = nil
		delete(s.staging, reqId)
	}
	s.m.Unlock()
	return
}

func (s *Stage) ShowStage() []StageInfo {
	s.m.Lock()
	defer s.m.Unlock()
	ret := make([]StageInfo, 0, len(s.staging))
	for k, r := range s.staging {
		ret = append(ret, StageInfo{k, (*r).Packet.Type})
	}
	return ret
}
