package server

import (
	"sync"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/uc"
)

type ControllerGroup struct {
	flow   *flow.Flow
	online map[uint16]*Controller
	toTun  chan []byte
	users  *uc.Users
	mutex  sync.RWMutex
}

func NewControllerGroup(f *flow.Flow, users *uc.Users, toTun chan []byte) *ControllerGroup {
	return &ControllerGroup{
		users:  users,
		online: make(map[uint16]*Controller),
		toTun:  toTun,
		flow:   f,
	}
}

func (c *ControllerGroup) RunDeliver(fromTun chan []byte) {
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
			ctl.WritePacket(d.Packet)
		case <-c.flow.IsClose():
			break loop
		}
	}
}

func (c *ControllerGroup) UserLogin(u *uc.User) *Controller {
	c.mutex.Lock()
	controller, ok := c.online[u.Id]
	if !ok {
		controller = NewController(c.flow, u, c.toTun)
		c.online[u.Id] = controller
	} else {
		controller.UserRelogin(u)
	}
	c.mutex.Unlock()
	return controller
}
