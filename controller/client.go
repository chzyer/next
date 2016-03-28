package controller

import (
	"encoding/json"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
)

type CliDelegate interface {
	OnNewDC(port []int)
}

type Client struct {
	*Controller
	toTun    chan<- []byte
	delegate CliDelegate
}

func NewClient(f *flow.Flow, delegate CliDelegate, toDC chan<- *packet.Packet, fromDC <-chan *packet.Packet, toTun chan<- []byte) *Client {
	ctl := NewController(f, toDC, fromDC)
	cli := &Client{
		Controller: ctl,
		toTun:      toTun,
		delegate:   delegate,
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
			switch pRecv.Type {
			case packet.DATA:
				select {
				case c.toTun <- pRecv.Data():
				case <-c.flow.IsClose():
					break loop
				}
			case packet.NEWDC:
				var port []int
				json.Unmarshal(pRecv.Payload, &port)
				if len(port) > 0 {
					c.delegate.OnNewDC(port)
				}
			}
			c.Send(pRecv.Reply(nil))
		case <-c.flow.IsClose():
			break loop
		}
	}
}
