package dchan

import (
	"fmt"
	"net"
	"sync"
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
	flow    *flow.Flow
	group   *Group
	session *packet.SessionIV
	mutex   sync.Mutex

	ports       []int
	toDC        <-chan *packet.Packet
	fromDC      chan<- *packet.Packet
	connectChan chan Slot
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

func (c *Client) Run() {
	c.group.Run()
	go c.loop()
}

func (c *Client) Close() {
	c.flow.Close()
}

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

	slot := Slot{
		Host: host,
		Port: uint16(port),
	}
	for i := 0; i < ChanCount; i++ {
		select {
		case c.connectChan <- slot:
		case <-c.flow.IsClose():
		}
	}
}

func (c *Client) onChanExit(slot Slot) {
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
loop:
	for {
		select {
		case slot := <-c.connectChan:
			err := c.MakeNewChannel(slot)
			if err != nil {
				time.Sleep(time.Second)
				// send back, TODO: prevent deadlock
				c.connectChan <- slot
				logex.Error(err)
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
