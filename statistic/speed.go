package statistic

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/chzyer/next/util"
)

type SpeedInfo struct {
	Current util.Unit
}

func (s *SpeedInfo) Merge(si *SpeedInfo) *SpeedInfo {
	s.Current += si.Current
	return s
}

type Speed struct {
	total      int64
	submitTime int64
	sync.Mutex
}

func NewSpeed() *Speed {
	return &Speed{}
}

func (s *Speed) updateLocked(n int64) {
	s.total = 0
	s.submitTime = n
}

func (s *Speed) checkOutdated() {
	now := time.Now().Unix()
	if now != atomic.LoadInt64(&s.submitTime) {
		s.Lock()
		if atomic.LoadInt64(&s.submitTime) != now {
			s.updateLocked(now)
		}
		s.Unlock()
	}
}

func (s *Speed) Submit(n int) {
	s.checkOutdated()
	atomic.AddInt64(&s.total, int64(n))
}

func (s *Speed) GetSpeed() *SpeedInfo {
	s.checkOutdated()
	return &SpeedInfo{
		Current: util.Unit(atomic.LoadInt64(&s.total)),
	}
}
