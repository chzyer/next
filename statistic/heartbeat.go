package statistic

import (
	"container/list"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

type HeartBeatInfo struct {
	total  time.Duration
	count  int64
	droped int64
}

func (s *HeartBeatInfo) dropStr() string {
	return fmt.Sprintf("%v", s.count)
}

func (s *HeartBeatInfo) rtt() time.Duration {
	if s.count == 0 {
		return 0
	}
	d := s.total / time.Duration(s.count)
	if d > time.Millisecond {
		d = d / time.Millisecond * time.Millisecond
	} else if d > time.Microsecond {
		d = d / time.Microsecond * time.Microsecond
	}
	return d
}

func (s *HeartBeatInfo) Add(s2 *HeartBeatInfo) {
	s.total += s2.total
	s.count += s2.count
	s.droped += s2.droped
}

type HeartBeat struct {
	start      time.Time
	lastTime   int               // the time of lastest slot
	slots      [90]HeartBeatInfo // 15 mintue, 10s one item
	lastCommit int64
	size       int
}

func (s *HeartBeat) getMin(n int) *HeartBeatInfo {
	h := &HeartBeatInfo{}
	n *= 6
	if n > s.size {
		n = s.size
	}
	for i := 0; i < n; i++ {
		h.Add(&s.slots[i])
	}
	return h
}

func (s *HeartBeat) getSlot() *HeartBeatInfo {
	now := time.Now()
	ts := (now.Minute() * 6) + (now.Second() / 10)

	if ts != s.lastTime {
		s.lastTime = ts
		for i := s.size - 1; i >= 1; i-- {
			s.slots[i] = s.slots[i-1]
		}
		s.slots[0] = HeartBeatInfo{}

		if s.size < len(s.slots) {
			s.size++
		}
	}
	if s.size == 0 {
		s.size = 1
	}
	return &s.slots[0]
}

func (s *HeartBeat) isNeedClean() error {
	stat := s.getMin(1)
	if stat.count > 10 {
		if stat.droped >= stat.count/2 {
			return fmt.Errorf("droped packets more than a half (%v>%v/2)", stat.droped, stat.count)
		}
	} else if stat.count > 5 && stat.droped == stat.count {
		return fmt.Errorf("too much droped packets")
	}
	// 60 seconds
	if time.Now().Unix()-atomic.LoadInt64(&s.lastCommit) > 10 {
		return fmt.Errorf("more than 60s no commit")
	}

	return nil
}

func (s *HeartBeat) submitDrop(n int) {
	slot := s.getSlot()
	atomic.StoreInt64(&s.lastCommit, time.Now().Unix())
	slot.droped += int64(n)
	slot.count++
}

func (s *HeartBeat) submitDuration(d time.Duration) {
	slot := s.getSlot()
	atomic.StoreInt64(&s.lastCommit, time.Now().Unix())
	slot.total += d
	slot.count++
}

func (s *HeartBeat) lifeTime() time.Duration {
	return time.Now().Round(time.Second).Sub(s.start.Round(time.Second))
}

func (s HeartBeat) String() string {
	min1 := s.getMin(1)
	min5 := s.getMin(5)
	min15 := s.getMin(15)
	return fmt.Sprintf("RTT: %v %v %v, LC: %v, LT: %v",
		min15.rtt(), min5.rtt(), min1.rtt(),
		(time.Second * time.Duration(time.Now().Unix()-s.lastCommit)).String(),
		s.lifeTime(),
	)
}

type heartBeatItem struct {
	reqid uint32
	time  time.Time
}

type HeartBeatStage struct {
	flow        *flow.Flow
	staging     *list.List
	receiveChan chan *packet.Packet
	addChan     chan *packet.Packet
	timeout     time.Duration
	delegate    CleanDelegate

	stat HeartBeat
}

type CleanDelegate interface {
	HeartBeatClean(error)
}

func NewHeartBeatStage(f *flow.Flow, timeout time.Duration, d CleanDelegate) *HeartBeatStage {
	hbs := &HeartBeatStage{
		timeout:     timeout,
		staging:     list.New(),
		receiveChan: make(chan *packet.Packet, 8),
		addChan:     make(chan *packet.Packet, 8),
		flow:        f,
		delegate:    d,
	}
	hbs.stat.start = time.Now()
	hbs.stat.lastCommit = time.Now().Unix()
	go hbs.loop()
	return hbs
}

func (h *HeartBeatStage) New() *packet.Packet {
	return packet.New(nil, packet.HEARTBEAT)
}

func (h *HeartBeatStage) Add(p *packet.Packet) {
	select {
	case h.addChan <- p:
	case <-h.flow.IsClose():
	}
}

func (h *HeartBeatStage) Receive(p *packet.Packet) {
	select {
	case h.receiveChan <- p:
	case <-h.flow.IsClose():
	}
}

func (h *HeartBeatStage) findElem(reqid uint32) *list.Element {
	now := time.Now()
	for elem := h.staging.Front(); elem != nil; elem = elem.Next() {
		if elem.Value.(heartBeatItem).reqid == reqid {
			return elem
		}

		if now.Sub(elem.Value.(heartBeatItem).time) > h.timeout {
			h.stat.submitDrop(1)
			h.staging.Remove(elem)
		}
	}
	return nil
}

func (h *HeartBeatStage) GetStat() *HeartBeat {
	s := h.stat
	return &s
}

func (h *HeartBeatStage) tryClean() bool {
	if err := h.stat.isNeedClean(); err != nil {
		h.delegate.HeartBeatClean(err)
		return true
	}
	return false
}

func (h *HeartBeatStage) item(elem *list.Element) heartBeatItem {
	return elem.Value.(heartBeatItem)
}

func (h *HeartBeatStage) GetLatency() (latency, lastCommit time.Duration) {
	n := int(time.Now().Unix() - h.stat.lastCommit)
	lastCommit = time.Duration(n) * time.Second
	info := h.stat.getMin(1)
	return info.rtt(), lastCommit
}

func (h *HeartBeatStage) loop() {
	ticker := time.NewTicker(h.timeout)
	defer ticker.Stop()
loop:
	for {
		select {
		case <-h.flow.IsClose():
			break loop
		case <-ticker.C:
			h.findElem(0) // just clean up
		case iv := <-h.receiveChan:
			elem := h.findElem(iv.ReqId)
			if elem == nil {
				// stat
				continue
			}
			h.stat.submitDuration(time.Now().Sub(h.item(elem).time))
			h.staging.Remove(elem)
		case iv := <-h.addChan:
			h.staging.PushBack(heartBeatItem{iv.ReqId, time.Now()})
		}
		if h.tryClean() {
			break loop
		}
	}
}
