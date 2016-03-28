package controller

import (
	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

type Client struct {
	*Controller
	toTun chan<- []byte
}

func NewClient(f *flow.Flow, toDC chan<- *packet.Packet, fromDC <-chan *packet.Packet, toTun chan<- []byte) *Client {
	ctl := NewController(f, toDC, fromDC)
	cli := &Client{
		Controller: ctl,
		toTun:      toTun,
	}
	go cli.loop()
	return cli
}

func (c *Client) loop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()
	out := c.GetOutChan()
loop:
	for {
		select {
		case pRecv := <-out:
			if pRecv.Type == packet.DATA {
				select {
				case c.toTun <- pRecv.Data():
				case <-c.flow.IsClose():
					break loop
				}
			}
		case <-c.flow.IsClose():
			break loop
		}
	}
}
