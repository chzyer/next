package server

import (
	"time"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/uc"
)

// one controller for one user
// read from out
// write to in
type Controller struct {
	flow      *flow.Flow
	u         *uc.User
	staging   map[uint32]*stage
	toTun     chan []byte
	writeChan chan *packet.Packet

	toUser   chan<- *packet.Packet
	fromUser <-chan *packet.Packet
}

type stage struct {
	p     *packet.Packet
	reply chan *packet.Packet
}

func NewController(f *flow.Flow, u *uc.User, toTun chan []byte) *Controller {
	fromUser, toUser := u.GetFromController()
	c := &Controller{
		u:     u,
		toTun: toTun,

		writeChan: make(chan *packet.Packet),
		toUser:    toUser,
		fromUser:  fromUser,
	}
	f.ForkTo(&c.flow, c.Close)
	go c.readLoop()
	go c.writeLoop()
	return c
}

func (c *Controller) readLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()
loop:
	for {
		select {
		case <-c.flow.IsClose():
			break loop
		case p := <-c.fromUser:
			if p.Type.IsResp() {
				if staging := c.staging[p.IV.ReqId]; staging != nil {
					select {
					case staging.reply <- p:
					default:
					}
				}
			} else {
				switch p.Type {
				case packet.Data:
					payload := p.Payload
					c.toTun <- payload
					// reply
					p.Type = packet.DataResp
					c.writeChan <- p
				default:
					logex.Error("unexpected packet type: ", p.Type)
				}
			}
		}
	}
}

func (c *Controller) writeLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

loop:
	for {
		select {
		case <-c.flow.IsClose():
			break loop
		case p := <-c.writeChan:
			c.toUser <- p
		}
	}
}
func (c *Controller) WritePacket(p *packet.Packet) {
	c.writeChan <- p
}

func (c *Controller) Write(p *packet.Packet) *packet.Packet {
	staging := c.staging[p.IV.ReqId]
	if staging == nil {
		staging = &stage{
			p:     p,
			reply: make(chan *packet.Packet),
		}
		c.staging[p.IV.ReqId] = staging
	}
	var reply *packet.Packet
loop:
	for {
		c.toUser <- staging.p
		select {
		case reply = <-staging.reply:
			break loop
		case <-time.After(time.Second):
		}
	}

	staging.p = nil
	return reply
}

func (c *Controller) UserRelogin(u *uc.User) {

}

func (c *Controller) Close() {
	c.flow.Close()
}
