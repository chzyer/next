package statistic

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/chzyer/next/util"
)

type SpeedInfo struct {
	Download util.Unit
	Upload   util.Unit
}

func (s *SpeedInfo) Merge(si *SpeedInfo) *SpeedInfo {
	s.Download += si.Download
	s.Upload += si.Upload
	return s
}

type Speed struct {
	upload     int64
	download   int64
	submitTime int64
	sync.Mutex
}

func NewSpeed() *Speed {
	return &Speed{}
}

func (s *Speed) updateLocked(n int64) {
	atomic.StoreInt64(&s.upload, 0)
	atomic.StoreInt64(&s.download, 0)
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

func (s *Speed) Upload(n int) {
	// s.checkOutdated()
	atomic.AddInt64(&s.upload, int64(n))
}

func (s *Speed) Download(n int) {
	// s.checkOutdated()
	atomic.AddInt64(&s.download, int64(n))
}

func (s *Speed) GetSpeed() *SpeedInfo {
	// s.checkOutdated()
	return &SpeedInfo{
		Download: util.Unit(atomic.SwapInt64(&s.download, 0)),
		Upload:   util.Unit(atomic.SwapInt64(&s.upload, 0)),
	}
}
