package packet

import (
	"container/list"
	"fmt"
	"time"

	"github.com/chzyer/flow"
)

type heartBeatStatMin struct {
	total  time.Duration
	count  int64
	droped int64
}

func (s *heartBeatStatMin) dropStr() string {
	return fmt.Sprintf("%v/%v", s.droped, s.count)
}

func (s *heartBeatStatMin) rtt() time.Duration {
	if s.count == 0 {
		return 0
	}
	d := s.total / time.Duration(s.count)
	d = d / time.Millisecond * time.Millisecond
	return d
}

func (s *heartBeatStatMin) Add(s2 *heartBeatStatMin) {
	s.total += s2.total
	s.count += s2.count
	s.droped += s2.droped
}

type HeartBeatStat struct {
	start    time.Time
	lastTime int
	slots    [30]heartBeatStatMin // 15 mintue, 30s one second
	size     int
}

func (s *HeartBeatStat) getMin(n int) *heartBeatStatMin {
	h := &heartBeatStatMin{}
	n *= 2
	if n > s.size {
		n = s.size
	}
	for i := 0; i < n; i++ {
		h.Add(&s.slots[i])
	}
	return h
}

func (s *HeartBeatStat) getSlot() *heartBeatStatMin {
	now := time.Now()
	ts := now.Minute() * 2
	if now.Second() > 30 {
		ts += 1
	}

	if ts != s.lastTime {
		s.lastTime = ts
		for i := s.size - 1; i >= 1; i-- {
			s.slots[i] = s.slots[i-1]
		}
		s.slots[0] = heartBeatStatMin{}

		if s.size < len(s.slots) {
			s.size++
		}
	}
	if s.size == 0 {
		s.size = 1
	}
	return &s.slots[0]
}

func (s *HeartBeatStat) needClean() error {
	stat := s.getMin(1)
	if stat.count > 10 {
		if stat.droped >= stat.count/2 {
			return fmt.Errorf("droped packets more than a half (%v>%v/2)", stat.droped, stat.count)
		}
	} else if stat.count > 5 && stat.droped == stat.count {
		return fmt.Errorf("too much droped packets")
	}
	min2 := s.getMin(2)
	min5 := s.getMin(5)
	if min2.rtt() > min5.rtt()*2 && min2.rtt() > 200*time.Millisecond {
		return fmt.Errorf("rtt in 2min(%v) more than 2 times of 5mins(%v)",
			min2.rtt(), min5.rtt(),
		)
	}
	return nil
}

func (s *HeartBeatStat) submitDrop(n int) {
	slot := s.getSlot()
	slot.droped += int64(n)
	slot.count++
}

func (s *HeartBeatStat) submitDuration(d time.Duration) {
	slot := s.getSlot()

	slot.total += d
	slot.count++
}

func (s *HeartBeatStat) lifeTime() time.Duration {
	return time.Now().Round(time.Second).Sub(s.start.Round(time.Second))
}

func (s HeartBeatStat) String() string {
	min1 := s.getMin(1)
	min5 := s.getMin(5)
	min15 := s.getMin(15)
	return fmt.Sprintf("avg: %v %v %v, drop: %v %v %v",
		min15.rtt(), min5.rtt(), min1.rtt(),
		min15.dropStr(), min5.dropStr(), min1.dropStr(),
	)
}

type heartBeatItem struct {
	reqid uint32
	time  time.Time
}

type HeartBeatStage struct {
	flow        *flow.Flow
	staging     *list.List
	receiveChan chan *IV
	addChan     chan *IV
	timeout     time.Duration
	clean       func(error)
	name        string

	stat HeartBeatStat
}

func NewHeartBeatStage(f *flow.Flow, timeout time.Duration, name string, clean func(error)) *HeartBeatStage {
	hbs := &HeartBeatStage{
		timeout:     timeout,
		staging:     list.New(),
		receiveChan: make(chan *IV, 8),
		addChan:     make(chan *IV, 8),
		name:        name,
		flow:        f,
		clean:       clean,
	}
	hbs.stat.start = time.Now()
	go hbs.loop()
	return hbs
}

func (h *HeartBeatStage) New() *Packet {
	return New(nil, HeartBeat)
}

func (h *HeartBeatStage) Add(iv *IV) {
	h.addChan <- iv
}

func (h *HeartBeatStage) Receive(iv *IV) {
	h.receiveChan <- iv
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

func (h *HeartBeatStage) GetStat() *HeartBeatStat {
	s := h.stat
	return &s
}

func (h *HeartBeatStage) tryClean() bool {
	if err := h.stat.needClean(); err != nil {
		h.clean(err)
		return true
	}
	return false
}

func (h *HeartBeatStage) item(elem *list.Element) heartBeatItem {
	return elem.Value.(heartBeatItem)
}

func (h *HeartBeatStage) loop() {
	ticker := time.NewTicker(10 * time.Second)
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
