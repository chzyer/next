package client

import (
	"github.com/chzyer/flow"
	"github.com/chzyer/next/util/clock"
)

type Client struct {
	cfg   *Config
	clock *clock.Clock
	flow  *flow.Flow
}

func New(cfg *Config, f *flow.Flow) *Client {
	cli := &Client{
		cfg:  cfg,
		flow: f,
	}
	return cli
}

func (c *Client) Run() {
	if err := c.initClock(); err != nil {
		c.flow.Error(err)
		return
	}
}
