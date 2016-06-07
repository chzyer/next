package controller

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/logex"
	"github.com/chzyer/next/packet"
)

var (
	ErrTimeout = fmt.Errorf("timed out")
)

type Controller struct {
	timeout time.Duration
	flow    *flow.Flow
	in      chan *Request
	out     packet.Chan
	toDC    packet.SendChan
	fromDC  packet.RecvChan
	reqId   uint32
	stage   *Stage

	cancelBroadcast *flow.Broadcast
}

func NewController(f *flow.Flow, toDC packet.SendChan, fromDC packet.RecvChan) *Controller {
	ctl := &Controller{
		timeout:         2 * time.Second,
		in:              make(chan *Request, 8),
		out:             make(packet.Chan),
		toDC:            toDC,
		fromDC:          fromDC,
		cancelBroadcast: flow.NewBroadcast(),
	}
	f.ForkTo(&ctl.flow, ctl.Close)
	ctl.stage = newStage()
	go ctl.readLoop()
	go ctl.writeLoop()
	go ctl.resendLoop()
	return ctl
}

func (c *Controller) CancelAll() {
	logex.Info("cancel all operation")
	c.cancelBroadcast.Notify()
}

func (c *Controller) GetOutChan() packet.RecvChan {
	return c.out.Recv()
}

func (c *Controller) GetReqId() uint32 {
	return atomic.AddUint32(&c.reqId, 1)
}

func (c *Controller) Close() {
	c.cancelBroadcast.Close()
	c.flow.Close()
}

func (c *Controller) WriteChan() chan *Request {
	return c.in
}

type Request struct {
	Packet  *packet.Packet
	Reply   chan *packet.Packet
	Timeout time.Duration
}

func NewRequest(p *packet.Packet, reply bool) *Request {
	req := &Request{Packet: p}
	if reply {
		req.Reply = make(chan *packet.Packet)
	}
	return req
}

func (c *Controller) send(req *Request) (*packet.Packet, error) {
	var timeout <-chan time.Time
	if req.Timeout > 0 {
		timeout = time.After(req.Timeout)
	}
	select {
	case c.in <- req:
		logex.Debug(req.Packet.Type.String())
		if req.Reply != nil {
			select {
			case rep := <-req.Reply:
				return rep, nil
			case <-c.flow.IsClose():
			}
		}
	case <-c.cancelBroadcast.Wait():
		return nil, flow.ErrCanceled
	case <-timeout:
		return nil, ErrTimeout
	case <-c.flow.IsClose():
	}
	return nil, nil
}

func (c *Controller) Request(req *packet.Packet) *packet.Packet {
	ret, _ := c.send(&Request{
		Packet: req,
		Reply:  make(chan *packet.Packet),
	})
	return ret
}

func (c *Controller) SendTimeout(req *packet.Packet, timeout time.Duration) bool {
	_, err := c.send(&Request{Packet: req, Timeout: timeout})
	return err != ErrTimeout
}

func (c *Controller) Send(req *packet.Packet) {
	c.send(&Request{Packet: req})
}

func (c *Controller) handlePacket(ps []*packet.Packet) bool {
	newPs := make([]*packet.Packet, 0, len(ps))
	for _, p := range ps {
		if p.Type.IsResp() {
			// println("I got Reply:", p.IV.ReqId)
			req := c.stage.Remove(p.ReqId)
			if req != nil && req.Reply != nil {
				select {
				case req.Reply <- p:
				default:
				}
			}
		}
		newPs = append(newPs, p)
	}

	// println("I need Reply to:", p.IV.ReqId)
	select {
	case c.out <- newPs:
	case <-c.flow.IsClose():
		return false
	}
	return true
}

func (c *Controller) readLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()
loop:
	for {
		select {
		case <-c.flow.IsClose():
			break loop
		case ps := <-c.fromDC:
			if !c.handlePacket(ps) {
				break loop
			}
		}
	}
}

func (c *Controller) resendLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

	ticker := time.NewTicker(c.timeout)
	defer ticker.Stop()
loop:
	for {
		switch c.flow.Tick(ticker) {
		case flow.F_CLOSED:
			break loop
		case flow.F_TIMEOUT:
		repop:
			req := c.stage.Pop(c.timeout)
			if req == nil {
				continue
			}
			logex.Debug("pop stage:", req.Packet.ReqId, req.Packet.Type.String())
			if req.Packet.Type == packet.DATA {
				continue
				// logex.Debug("resend:", req.Packet.ReqId, req.Packet.Type.String())
			} else {
				logex.Info("resend:", req.Packet.ReqId, req.Packet.Type.String())
			}
			select {
			case c.in <- req:
				goto repop
			case <-c.flow.IsClose():
				break loop
			}
		}
	}
}

func (c *Controller) writeLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

	var bufferPackets []*packet.Packet
	timer := time.NewTimer(time.Millisecond)
	timer.Stop()

loop:
	for {
		select {
		case <-c.flow.IsClose():
			break loop
		case req := <-c.in:
			// add to staging
			if req.Packet.Type.IsReq() {
				req.Packet.SetReqId(c)
				c.stage.Add(req)
			}
			bufferPackets = append(bufferPackets, req.Packet)

			timer.Reset(time.Millisecond)
		buffering:
			for {
				select {
				case req := <-c.in:
					if req.Packet.Type.IsReq() {
						req.Packet.SetReqId(c)
						c.stage.Add(req)
					}
					bufferPackets = append(bufferPackets, req.Packet)
				case <-timer.C:
					break buffering
				}
			}

			// do buffer
			select {
			case c.toDC <- bufferPackets:
				bufferPackets = nil
			case <-c.flow.IsClose():
				break loop
			}
		}
	}
}

func (c *Controller) ShowStage() []StageInfo {
	return c.stage.ShowStage()
}
