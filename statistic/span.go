package statistic

import (
	"sync/atomic"
	"time"
)

type Span struct {
	duration int64
	count    int32
	stop     chan struct{}
}

func NewSpan() *Span {
	return &Span{
		stop: make(chan struct{}),
	}
}

func (s *Span) Submit(d time.Duration) {
	atomic.AddInt64(&s.duration, int64(d))
	atomic.AddInt32(&s.count, 1)
}

func (s *Span) Stop() {
	close(s.stop)
}

func (s *Span) Tick() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.printResult()
			case <-s.stop:
				s.printResult()
				return
			}
		}
	}()
}

func (s *Span) printResult() {
	total := time.Duration(atomic.SwapInt64(&s.duration, 0))
	count := atomic.SwapInt32(&s.count, 0)
	if count > 0 {
		total /= time.Duration(count)
	}
	println(total.String())
}
