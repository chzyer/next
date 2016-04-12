package dchan

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/datachannel"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/util"
	"gopkg.in/logex.v1"
)

const (
	ChanCount = 2
)

type Slot struct {
	Host string
	Port uint16
}

type Client struct {
	flow         *flow.Flow
	group        *Group
	session      *packet.SessionIV
	mutex        sync.Mutex
	runningChans int32

	ports        []int
	toDC         <-chan *packet.Packet
	fromDC       chan<- *packet.Packet
	connectChan  chan Slot
	onAllBackoff func()
}

// out is which datachannel can write for
// all of channel share on fromDC, and have their owned toDC
// client receive all packet from toDC and try to send them
func NewClient(f *flow.Flow, s *packet.SessionIV,
	toDC <-chan *packet.Packet, fromDC chan<- *packet.Packet) *Client {

	cli := &Client{
		connectChan: make(chan Slot, 1024),
		session:     s,
		toDC:        toDC,
		fromDC:      fromDC,
	}
	f.ForkTo(&cli.flow, cli.Close)
	cli.group = NewGroup(cli.flow)
	return cli
}

func (c *Client) CloseChannel(src, dst string) error {
	return c.group.CloseChannel(src, dst)
}

func (c *Client) Run() {
	c.group.Run()
	go c.loop()
}

func (c *Client) Close() {
	c.flow.Close()
}

// AddHost will exclude endpoint which is already exists
func (c *Client) AddHost(host string, port int) {
	c.mutex.Lock()
	added := util.InInts(int(port), c.ports)
	if !added {
		c.ports = append(c.ports, int(port))
	}
	c.mutex.Unlock()
	if added {
		return
	}
	logex.Infof("add new endpoint: %v:%v", host, port)

	slot := Slot{
		Host: host,
		Port: uint16(port),
	}
	for i := 0; i < ChanCount; i++ {
		select {
		case c.connectChan <- slot:
		case <-c.flow.IsClose():
			logex.Info("flow is closed, ignore AddHost")
		}
	}
}

func (c *Client) Ports() []int {
	return c.ports
}

func (c *Client) GetRunningChans() int {
	return int(atomic.LoadInt32(&c.runningChans))
}

// used by MakeNewChannel
func (c *Client) onChanExit(slot Slot) {
	newRunning := atomic.AddInt32(&c.runningChans, -1)
	if newRunning == 0 {
		if c.onAllBackoff != nil {
			go c.onAllBackoff()
		}
	}
	select {
	case c.connectChan <- slot:
	case <-c.flow.IsClose():
	}
}

func (c *Client) MakeNewChannel(slot Slot) error {
	host := fmt.Sprintf("%v:%v", slot.Host, slot.Port)
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return logex.Trace(err)
	}
	session := c.session.Clone(slot.Port)
	ch := NewChannel(c.flow, session, conn, c.fromDC)
	if err := datachannel.ClientCheckAuth(conn, session); err != nil {
		return logex.Trace(err)
	}
	ch.AddOnClose(func() {
		c.onChanExit(slot)
	})
	c.group.AddWithAutoRemove(ch)
	ch.Run()
	return nil
}

func (c *Client) loop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

loop:
	for !c.flow.IsClosed() {
		select {
		case slot := <-c.connectChan:
			logex.Infof("prepare to connect to %v:%v", slot.Host, slot.Port)
			err := c.MakeNewChannel(slot)
			if err != nil {
				logex.Error(err, ",wait 1 second")
				time.Sleep(time.Second)
				// send back, TODO: prevent deadlock
				select {
				case c.connectChan <- slot:
					logex.Info("resend back to channel")
				case <-c.flow.IsClose():
					break loop
				}
			} else {
				atomic.AddInt32(&c.runningChans, 1)
			}
		case p := <-c.toDC:
			logex.Debug(p)
			c.group.Send(p)
		case <-c.flow.IsClose():
			break loop
		}
	}
}

func (c *Client) GetUsefulChan() []*Channel {
	return c.group.GetUsefulChan()
}

func (c *Client) GetStats() string {
	return c.group.GetStatsInfo()
}

func (c *Client) UpdateRemoteAddrs(host string, ports []int) {
	for _, p := range ports {
		c.AddHost(host, p)
	}
}

func (c *Client) SetOnAllBackoff(f func()) {
	c.onAllBackoff = f
}
