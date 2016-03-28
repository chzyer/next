package datachannel

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/util"
)

const (
	DataChannelSize = 4
)

type dcSlot struct {
	Addr        string
	On          bool
	Port        uint16
	backoffTime time.Time
	dc          *DC
}

func newDcSlot(host string, port int) dcSlot {
	return dcSlot{
		Addr: host,
		Port: uint16(port),
	}
}

type Client struct {
	slots   []dcSlot
	ports   []int
	host    string
	flow    *flow.Flow
	session *packet.SessionIV
	running int32

	mutex sync.Mutex

	in           chan *packet.Packet
	out          chan *packet.Packet
	dcExit       chan int
	onAllBackoff func()
	kickFire     chan struct{}
}

func NewClient(f *flow.Flow, host string, port int, s *packet.SessionIV,
	dcIn, dcOut chan *packet.Packet) *Client {

	dc := &Client{
		session: s,
		host:    host,
		ports:   []int{port},

		in:       dcIn,
		out:      dcOut,
		dcExit:   make(chan int),
		kickFire: make(chan struct{}, 1),
	}
	f.ForkTo(&dc.flow, dc.Close)
	dc.slots = dc.makeSlots(dc.ports, DataChannelSize)
	go dc.loop()
	return dc
}

func (d *Client) Close() {
	d.flow.Close()
}

func (d *Client) SetOnAllChannelsBackoff(f func()) {
	d.onAllBackoff = f
}

func (d *Client) loop() {
	for {
		d.mutex.Lock()
		slotIdx, wait := d.findOff()
		d.mutex.Unlock()
		if slotIdx < 0 {
			select {
			case <-d.kickFire:
			case <-time.After(wait):
			case <-d.flow.IsClose():
				return
			}
			continue
		}

		d.mutex.Lock()
		_, err := d.newDC(slotIdx)
		d.mutex.Unlock()
		if err != nil {
			logex.Error(err, d.running)
		}
	}
}

func (d *Client) findOff() (int, time.Duration) {
	now := time.Now()
	var wait time.Duration
	for idx := range d.slots {
		if !d.slots[idx].On {
			duration := d.slots[idx].backoffTime.Sub(now)
			if duration <= 0 {
				return idx, 0
			} else {
				if wait == 0 {
					wait = duration
				} else if duration < wait {
					wait = duration
				}
			}
		}
	}
	if wait == 0 {
		wait = time.Minute
	}
	return -1, wait
}

func (d *Client) makeSlots(ports []int, size int) []dcSlot {
	slots := make([]dcSlot, 0, len(ports)*size)
	for i := 0; i < size; i++ {
		for _, port := range ports {
			slots = append(slots, newDcSlot(d.host, port))
		}
	}
	return slots
}

func (d *Client) compareSlots(ports []int) (remain []int, newp []int) {
	for _, p := range d.ports {
		if util.IntIn(p, ports) {
			remain = append(remain, p)
		}
	}
	for _, p := range ports {
		if !util.IntIn(p, remain) {
			newp = append(newp, p)
		}
	}
	return
}

func (d *Client) makeNewSlots(ports []int) {
	remain, newp := d.compareSlots(ports)
	// not new and not delete
	if len(newp) == 0 && len(remain) == len(d.ports) {
		return
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()
	newSlots := make([]dcSlot, 0, (len(newp)+len(remain))*DataChannelSize)

	for _, slot := range d.slots {
		if util.IntIn(int(slot.Port), remain) {
			newSlots = append(newSlots, slot)
		}
	}
	for _, p := range newp {
		for i := 0; i < DataChannelSize; i++ {
			newSlots = append(newSlots, newDcSlot(d.host, p))
		}
	}
	d.slots = newSlots
}

func (d *Client) UpdateRemoteAddrs(ports []int) {
	logex.Info("new client", ports)
	d.makeNewSlots(ports)

	select {
	case d.kickFire <- struct{}{}:
	default:
	}
}

// inlock
func (d *Client) newDC(idx int) (*DC, error) {

	host := d.slots[idx].Addr
	port := d.slots[idx].Port
	endpoint := fmt.Sprintf("%v:%v", host, port)
	dc, err := DialDC(endpoint, d.flow, d.session.Clone(port),
		d.onDataChannelExits(idx), d.in, d.out)
	if err != nil {
		d.slots[idx].backoffTime = time.Now().Add(10 * time.Second)
		return nil, logex.Trace(err)
	}
	atomic.AddInt32(&d.running, 1)
	d.slots[idx].On = true
	d.slots[idx].dc = dc
	logex.Info("new datachannel to", host)
	return dc, nil
}

func (d *Client) GetStats() string {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	buf := bytes.NewBuffer(nil)
	for idx := range d.slots {
		dc := d.slots[idx].dc
		if dc != nil {
			buf.WriteString(dc.Name() + ": " + dc.GetStat().String() + "\n")
		}
	}
	return buf.String()
}

func (d *Client) onDataChannelExits(idx int) func() {
	return func() {
		d.mutex.Lock()
		d.slots[idx].On = false
		d.slots[idx].dc = nil
		if atomic.AddInt32(&d.running, -1) == 0 {
			if d.onAllBackoff != nil {
				d.onAllBackoff()
			}
		}
		d.mutex.Unlock()
		select {
		case d.kickFire <- struct{}{}:
		default:
		}
	}
}
