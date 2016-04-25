package dchan

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/statistic"
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

func (s Slot) String() string {
	return fmt.Sprintf("%v:%v", s.Host, s.Port)
}

type ClientDelegate interface {
	OnAllBackoff(*Client)
	OnLinkRefused()
}

type Client struct {
	flow         *flow.Flow
	group        *Group
	session      *packet.Session
	mutex        sync.Mutex
	runningChans int32
	chanFactory  ChannelFactory

	delegate ClientDelegate

	ports       []int
	fromDC      chan<- *packet.Packet
	connectChan chan Slot
}

// out is which datachannel can write for
// all of channel share on fromDC, and have their owned toDC
// client receive all packet from toDC and try to send them
func NewClient(f *flow.Flow,
	s *packet.Session, delegate ClientDelegate, chanTyp string,
	toDC <-chan *packet.Packet, fromDC chan<- *packet.Packet) (*Client, error) {

	if err := CheckType(chanTyp); err != nil {
		return nil, logex.Trace(err)
	}
	cli := &Client{
		delegate:    delegate,
		connectChan: make(chan Slot, 1024),
		session:     s,
		fromDC:      fromDC,
		chanFactory: GetChannelType(chanTyp),
	}
	f.ForkTo(&cli.flow, cli.Close)
	cli.group = NewGroup(cli.flow, toDC, fromDC)
	return cli, nil
}

func (c *Client) CloseChannel(name string) error {
	return c.group.CloseChannel(name)
}

func (c *Client) Run() {
	c.group.Run()
	go c.connectLoop()
}

func (c *Client) GetFlow() *flow.Flow {
	return c.flow
}

func (c *Client) Close() {
	if !c.flow.MarkExit() {
		return
	}
	c.flow.Close()
	logex.Info("closed")
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

func (c *Client) tryToCallBackoff() bool {
	if atomic.LoadInt32(&c.runningChans) == 0 {
		c.callOnAllBackoff()
		return true
	}
	return false
}

func (c *Client) callOnAllBackoff() {
	if c.flow.IsExit() {
		return
	}
	c.delegate.OnAllBackoff(c)
}

// used by MakeNewChannel
func (c *Client) onChanExit(slot Slot) {
	newRunning := atomic.AddInt32(&c.runningChans, -1)
	if newRunning == 0 {
		c.callOnAllBackoff()
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
	session := c.session.Clone()
	ch := c.chanFactory.NewClient(c.flow, session, conn, c.fromDC)
	ch.AddOnClose(func() {
		c.onChanExit(slot)
	})
	c.group.AddWithAutoRemove(ch)
	ch.Run()
	return nil
}

func (c *Client) GetSpeedInfo() *statistic.SpeedInfo {
	return c.group.GetSpeed()
}

func (c *Client) connectLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

	getTimeSegment := func() int {
		return time.Now().Second() / 10
	}

	generation := 0
	timeSegment := getTimeSegment()
	connectCount := 0
	waitTime := time.Second

loop:
	for !c.flow.IsClosed() {
		select {
		case slot := <-c.connectChan:
			if getTimeSegment() != timeSegment {
				connectCount = 1
				generation++
			} else {
				connectCount++
			}

			if connectCount > 5 && generation > 1 {
				waitTime = 10 * time.Second
			} else {
				waitTime = time.Second
			}

			logex.Debugf("prepare to connect to %v:%v", slot.Host, slot.Port)
			err := c.MakeNewChannel(slot)
			if err != nil {
				if strings.Contains(err.Error(), "connection refused") {
					logex.Error("connect to", slot, "refused, remove it")
					c.delegate.OnLinkRefused()
					c.tryToCallBackoff()
					continue
				}
				logex.Error(err, ",wait", waitTime)
				time.Sleep(waitTime)
				// send back, TODO: prevent deadlock
				select {
				case c.connectChan <- slot:
					if c.tryToCallBackoff() {
						continue
					}
				case <-c.flow.IsClose():
					break loop
				}
			} else {
				atomic.AddInt32(&c.runningChans, 1)
			}
		case <-c.flow.IsClose():
			break loop
		}
	}
}

func (c *Client) GetUsefulChan() []Channel {
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
