package dchan

import (
	"bytes"
	"container/list"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/statistic"
	"github.com/chzyer/next/util"
)

// Channel can't close by Group
type Group struct {
	flow          *flow.Flow
	chanList      *list.List
	chanListGuard sync.RWMutex

	onNewUsefulChan  chan struct{}
	onNewUsefullCase reflect.SelectCase
	flowIsCloseCase  reflect.SelectCase

	usefulChans atomic.Value // []int
	selectCase  []reflect.SelectCase
}

func NewGroup(f *flow.Flow) *Group {
	newUseful := make(chan struct{}, 1)
	g := &Group{
		chanList:        list.New(),
		onNewUsefulChan: newUseful,
		onNewUsefullCase: reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(newUseful),
		},
	}
	f.ForkTo(&g.flow, g.Close)
	g.flowIsCloseCase = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(f.IsClose()),
	}
	return g
}

func (g *Group) findChannel(f func(Channel) bool) Channel {
	var ret Channel
	g.chanListGuard.RLock()
	for elem := g.chanList.Front(); elem != nil; elem = elem.Next() {
		if f(elem.Value.(Channel)) {
			ret = elem.Value.(Channel)
			break
		}
	}
	g.chanListGuard.RUnlock()
	return ret
}

func (g *Group) CloseChannel(name string) error {
	ch := g.findChannel(func(c Channel) bool {
		return c.Name() == name
	})
	if ch == nil {
		return fmt.Errorf("channel is not found")
	}
	ch.Close()
	return nil
}

func (g *Group) ChannelCount() int {
	g.chanListGuard.RLock()
	count := g.chanList.Len()
	g.chanListGuard.RUnlock()
	return count
}

func (g *Group) GetUsefulChan() []Channel {
	g.chanListGuard.RLock()
	defer g.chanListGuard.RUnlock()

	var ret []Channel
	usefuls := g.GetUseful()
	idx := 0
	for elem := g.chanList.Front(); elem != nil; elem = elem.Next() {
		if util.InInts(idx, usefuls) {
			ret = append(ret, elem.Value.(Channel))
		}
		idx++
	}
	return ret
}

func (g *Group) GetStatsInfo() string {
	buf := bytes.NewBuffer(nil)
	g.findChannel(func(ch Channel) bool {
		buf.WriteString(fmt.Sprintf("%v: %v\n",
			ch.Name(), ch.GetStat().String(),
		))
		return false
	})
	return buf.String()
}

func (g *Group) Run() {
	go g.loop()
}

func (g *Group) loop() {
	g.flow.Add(1)
	defer g.flow.DoneAndClose()

	usefulTick := time.NewTicker(5 * time.Second)
	defer usefulTick.Stop()

loop:
	for {
		select {
		case <-usefulTick.C:
			g.chanListGuard.Lock()
			g.updateUsefulLocked()
			g.chanListGuard.Unlock()
		case <-g.flow.IsClose():
			break loop
		}
	}
}

type latencies struct {
	Latency time.Duration
	Idx     int
}

func (g *Group) findUsefulLocked() []int {
	idx := 0
	infos := make([]*latencies, 0, g.chanList.Len())
	var minLatency, maxLatency time.Duration
	for elem := g.chanList.Front(); elem != nil; elem = elem.Next() {
		ch := elem.Value.(Channel)
		latency, lastCommit := ch.Latency()
		if lastCommit >= 5*time.Second {
			continue
		}
		infos = append(infos, &latencies{
			Idx:     idx,
			Latency: latency,
		})
		idx++

		// channel which is not heartbeat yet
		if latency == 0 {
			continue
		}
		if minLatency > latency || minLatency == 0 {
			minLatency = latency
		}
		if maxLatency < latency {
			maxLatency = latency
		}
	}

	ret := make([]int, 0, len(infos))
	// we have no choise
	meanVal := (minLatency + maxLatency) / 2
	for _, info := range infos {
		if info.Latency <= meanVal || len(infos) <= 2 {
			ret = append(ret, info.Idx)
		}
	}
	return ret
}

func (g *Group) updateUsefulLocked() {
	useful := g.findUsefulLocked()
	old := g.GetUseful()
	g.usefulChans.Store(useful)
	// notify
	if !util.EqualInts(useful, old) {
		select {
		case g.onNewUsefulChan <- struct{}{}:
		default:
		}
	}
}

func (g *Group) GetUseful() []int {
	useful := g.usefulChans.Load()
	if useful == nil {
		return nil
	}
	return useful.([]int)
}

func (g *Group) Send(p *packet.Packet) {
	pv := reflect.ValueOf(p)
resend:
	g.chanListGuard.RLock()
	usefulChans := g.GetUseful()
	selectCase := make([]reflect.SelectCase, len(usefulChans)+2)
	for caseIdx, chanIdx := range usefulChans {
		selectCase[caseIdx] = g.selectCase[chanIdx]
		selectCase[caseIdx].Send = pv
	}
	g.chanListGuard.RUnlock()

	// case <-g.flow.IsClosed()
	selectCase[len(selectCase)-2] = g.flowIsCloseCase
	// case <-g.onNewUsefulChan:  notify if we got a new chose
	selectCase[len(selectCase)-1] = g.onNewUsefullCase
	// TODO: how about all of this is fail?
	chosen, _, _ := reflect.Select(selectCase)
	if chosen == len(selectCase)-1 {
		goto resend
	}
}

func (g *Group) AddWithAutoRemove(c Channel) {
	logex.Info("new channel:", c.Name())
	g.chanListGuard.Lock()
	elem := g.chanList.PushFront(c)
	g.makeSelectCaseLocked()
	g.updateUsefulLocked()
	g.chanListGuard.Unlock()

	c.AddOnClose(func() {
		logex.Info("remove channel:", c.Name())
		g.chanListGuard.Lock()
		g.chanList.Remove(elem)
		g.makeSelectCaseLocked()
		g.updateUsefulLocked()
		g.chanListGuard.Unlock()
	})

}

func (g *Group) GetSpeed() *statistic.SpeedInfo {
	var s statistic.SpeedInfo
	g.findChannel(func(ch Channel) bool {
		s.Merge(ch.GetSpeed())
		return false
	})
	return &s
}

func (g *Group) makeSelectCaseLocked() {
	g.selectCase = make([]reflect.SelectCase, g.chanList.Len())
	idx := 0
	for elem := g.chanList.Front(); elem != nil; elem = elem.Next() {
		g.selectCase[idx] = reflect.SelectCase{
			Dir:  reflect.SelectSend,
			Chan: reflect.ValueOf(elem.Value.(Channel).ChanWrite()),
		}
		idx++
	}
}

func (g *Group) Close() {
	g.flow.Close()
}
