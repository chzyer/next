package packet

import (
	"container/list"
	"fmt"
	"time"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
)

type HeartBeatStat struct {
	total  time.Duration
	count  int64
	droped int64
}

func (s HeartBeatStat) String() string {
	return fmt.Sprintf(
		"avg: %v, droped: %.2f",
		s.total/time.Duration(s.count),
		float64(s.droped*100/s.count),
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

	stat HeartBeatStat
}

func NewHeartBeatStage(f *flow.Flow, timeout time.Duration) *HeartBeatStage {
	hbs := &HeartBeatStage{
		timeout:     timeout,
		staging:     list.New(),
		receiveChan: make(chan *IV, 8),
		addChan:     make(chan *IV, 8),
		flow:        f,
	}
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
			h.stat.droped++
			h.stat.count++
			h.staging.Remove(elem)
		}
	}
	return nil
}

func (h *HeartBeatStage) GetStat() *HeartBeatStat {
	s := h.stat
	return &s
}

func (h *HeartBeatStage) submitDuration(d time.Duration) {
	h.stat.total += d
	h.stat.count++
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
			logex.Info(h.GetStat())
		case iv := <-h.receiveChan:
			elem := h.findElem(iv.ReqId)
			if elem == nil {
				// stat
				continue
			}
			h.submitDuration(time.Now().Sub(h.item(elem).time))
			h.staging.Remove(elem)
		case iv := <-h.addChan:
			h.staging.PushBack(heartBeatItem{iv.ReqId, time.Now()})
		}
	}
}
