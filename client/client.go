package client

import (
	"github.com/chzyer/flow"
	"github.com/chzyer/next/ip"
	"github.com/chzyer/next/packet"
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
	*f.Debug = cfg.Debug
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

	port := 13111
	session := packet.NewSessionIV(
		uint16(remoteCfg.UserId), uint16(port), []byte(remoteCfg.Token))
	in := make(chan *packet.Packet)
	out := make(chan *packet.Packet)

	dc, err := NewDataChannel("nexts", c.flow.Fork(0), session, in, out)
	if err != nil {
		c.flow.Error(err)
		return
	}
	println("ok")

	_ = dc
	_ = tun

}
