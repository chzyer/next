package controller

import (
	"encoding/json"

	"github.com/chzyer/flow"
	"github.com/chzyer/logex"
	"github.com/chzyer/next/packet"
)

type CliDelegate interface {
	OnNewDC(port []int)
}

type Client struct {
	*Controller
	toTun    chan<- []byte
	newDC    chan struct{}
	delegate CliDelegate
}

func NewClient(f *flow.Flow, delegate CliDelegate, toDC packet.SendChan, fromDC packet.RecvChan, toTun chan<- []byte) *Client {
	ctl := NewController(f, toDC, fromDC)
	cli := &Client{
		Controller: ctl,
		toTun:      toTun,
		delegate:   delegate,
		newDC:      make(chan struct{}, 1),
	}
	go cli.loop()
	go cli.requestDCLoop()
	return cli
}

func (c *Client) GetFlow() *flow.Flow {
	return c.flow
}

func (c *Client) RequestNewDC() {
	logex.Info("request new dc")
	select {
	case c.newDC <- struct{}{}:
	default:
	}
}

func (c *Client) requestDCLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

loop:
	for {
		select {
		case <-c.newDC:
			c.Send(packet.New(nil, packet.NEWDC))
		case <-c.flow.IsClose():
			break loop
		}
	}
}

func (c *Client) handlePacket(p *packet.Packet) bool {
	switch p.Type {
	case packet.DATA:
		select {
		case c.toTun <- p.Payload():
		case <-c.flow.IsClose():
			return false
		}
	case packet.NEWDC_R:
		var port []int
		json.Unmarshal(p.Payload(), &port)
		if len(port) > 0 {
			c.delegate.OnNewDC(port)
		}
	}
	if p.Type.IsReq() {
		c.Send(p.Reply(nil))
	}
	return true
}

func (c *Client) loop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()
	out := c.GetOutChan()
loop:
	for {
		select {
		case pRecv := <-out:
			for _, p := range pRecv {
				if !c.handlePacket(p) {
					break loop
				}
			}
		case <-c.flow.IsClose():
			break loop
		}
	}
}
