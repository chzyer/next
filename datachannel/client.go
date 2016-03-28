package datachannel

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
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

func newDcSlot(host string) dcSlot {
	idx := strings.Index(host, ":")
	port, err := strconv.Atoi(host[idx+1:])
	if err != nil {
		panic(err)
	}
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
	hosts := []string{fmt.Sprintf("%v:%v", host, port)}
	dc.slots = dc.makeSlots(hosts, DataChannelSize)
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
		slotIdx, wait := d.findOff()
		if slotIdx < 0 {
			select {
			case <-d.kickFire:
			case <-time.After(wait):
			case <-d.flow.IsClose():
				return
			}
			continue
		}
		_, err := d.newDC(slotIdx)
		if err != nil {
			logex.Error(err, d.running)
			continue
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

func (d *Client) makeSlots(remoteAddrs []string, size int) []dcSlot {
	slots := make([]dcSlot, 0, len(remoteAddrs)*size)
	for i := 0; i < size; i++ {
		for _, addr := range remoteAddrs {
			slots = append(slots, newDcSlot(addr))
		}
	}
	return slots
}

func (d *Client) UpdateRemoteAddrs(ports []int) {
	select {
	case d.kickFire <- struct{}{}:
	default:
	}
}

func (d *Client) newDC(idx int) (*DC, error) {
	host := d.slots[idx].Addr
	port := d.slots[idx].Port
	dc, err := DialDC(host, d.flow, d.session.Clone(port),
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
		d.slots[idx].On = false
		d.slots[idx].dc = nil
		if atomic.AddInt32(&d.running, -1) == 0 {
			if d.onAllBackoff != nil {
				d.onAllBackoff()
			}
		}
		select {
		case d.kickFire <- struct{}{}:
		default:
		}
	}
}
