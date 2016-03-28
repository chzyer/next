package controller

import (
	"sync"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/uc"
	"gopkg.in/logex.v1"
)

type SvrDelegate interface {
	GetAllDataChannel() []int
}

type Group struct {
	delegate SvrDelegate
	flow     *flow.Flow
	online   map[uint16]*Server
	toTun    chan<- []byte
	users    *uc.Users
	mutex    sync.RWMutex
}

func NewGroup(f *flow.Flow, delegate SvrDelegate, users *uc.Users, toTun chan<- []byte) *Group {
	return &Group{
		delegate: delegate,
		users:    users,
		online:   make(map[uint16]*Server),
		toTun:    toTun,
		flow:     f,
	}
}

func (c *Group) RunDeliver(fromTun <-chan []byte) {
loop:
	for {
		select {
		case ipPacket := <-fromTun:
			d := packet.NewDataPacket(ipPacket)
			u := c.users.FindByIP(d.DestIP())
			if u == nil {
				logex.Errorf("user not found: %v", d.DestIP())
				continue
			}
			c.mutex.RLock()
			ctl := c.online[u.Id]
			c.mutex.RUnlock()
			ctl.Send(d.Packet)
		case <-c.flow.IsClose():
			break loop
		}
	}
}

func (c *Group) UserLogin(u *uc.User) *Server {
	c.mutex.Lock()
	controller, ok := c.online[u.Id]
	if !ok {
		controller = NewServer(c.flow, u, c.toTun)
		c.online[u.Id] = controller
	} else {
		controller.UserRelogin(u)
	}
	c.mutex.Unlock()
	controller.NotifyDataChannel(c.delegate.GetAllDataChannel())
	return controller
}
