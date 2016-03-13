package client

import (
	"github.com/chzyer/flow"
	"github.com/chzyer/next/ip"
	"github.com/chzyer/next/tunnel"
	"github.com/chzyer/next/util/clock"
	"gopkg.in/logex.v1"
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

	remoteCfg, err := c.Login(c.cfg.UserName, c.cfg.Password)
	if err != nil {
		c.flow.Error(err)
		return
	}
	logex.Struct(remoteCfg)

	ipnet, err := ip.ParseCIDR(remoteCfg.Gateway)
	if err != nil {
		c.flow.Error(err)
		return
	}
	ipnet.IP = ip.ParseIP(remoteCfg.INet)
	tun, err := tunnel.New(&tunnel.Config{
		DevId:   c.cfg.DevId,
		Gateway: ipnet.ToNet(),
		MTU:     remoteCfg.MTU,
		Debug:   c.cfg.Debug,
	})
	if err != nil {
		c.flow.Error(err)
		return
	}
	_ = tun

}
