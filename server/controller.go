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
	in        chan *packet.Packet
	out       chan *packet.Packet
	staging   map[uint32]*stage
	toTun     chan []byte
	writeChan chan *packet.Packet
}

type stage struct {
	p     *packet.Packet
	reply chan *packet.Packet
}

func NewController(f *flow.Flow, u *uc.User, toTun chan []byte) *Controller {
	in, out := u.GetChannel()
	c := &Controller{
		u:         u,
		in:        in,
		out:       out,
		toTun:     toTun,
		writeChan: make(chan *packet.Packet),
	}
	f.ForkTo(&c.flow, c.Close)
	return c
}

func (c *Controller) readLoop() {
	for {
		select {
		case p := <-c.out:
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
loop:
	for {
		select {
		case <-c.flow.IsClose():
			break loop
		case p := <-c.writeChan:
			c.in <- p
		}
	}
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
		c.in <- staging.p
		select {
		case reply = <-staging.reply:
			break loop
		case <-time.After(time.Second):
		}
	}

	staging.p = nil
	return reply
}

func (c *Controller) Close() {

}
