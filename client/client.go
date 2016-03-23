package client

import (
	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/uc"
	"github.com/chzyer/next/util/clock"
)

type Client struct {
	cfg   *Config
	clock *clock.Clock
	flow  *flow.Flow
	tun   *Tun
}

func New(cfg *Config, f *flow.Flow) *Client {
	cli := &Client{
		cfg:  cfg,
		flow: f,
	}
	return cli
}

func (c *Client) Close() {
	c.flow.Close()
}

func (c *Client) initDataChannel(remoteCfg *uc.AuthResponse) (in, out chan *packet.Packet, err error) {
	port := remoteCfg.GetDataChannelPort()
	session := packet.NewSessionIV(
		uint16(remoteCfg.UserId), uint16(port), []byte(remoteCfg.Token))

	in = make(chan *packet.Packet)
	out = make(chan *packet.Packet)
	dc, err := NewDataChannel(
		remoteCfg.DataChannel, c.flow, session, in, out)
	if err != nil {
		return nil, nil, err
	}
	_ = dc

	return in, out, nil
}

func (c *Client) initTun(remoteCfg *uc.AuthResponse) (in, out chan []byte, err error) {
	in = make(chan []byte)
	out = make(chan []byte)
	tun, err := newTun(c.flow, remoteCfg, c.cfg)
	if err != nil {
		return nil, nil, err
	}
	tun.Run(in, out)
	return in, out, nil
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

	dcIn, dcOut, err := c.initDataChannel(remoteCfg)
	if err != nil {
		c.flow.Error(err)
		return
	}

	tunIn, tunOut, err := c.initTun(remoteCfg)
	if err != nil {
		c.flow.Error(err)
		return
	}

	go func() {
	loop:
		for {
			select {
			case <-c.flow.IsClose():
				break loop
			case data := <-tunOut:
				dcIn <- packet.New(data, packet.Data)
			case pRecv := <-dcOut:
				if pRecv.Type == packet.Data {
					tunIn <- pRecv.Data()
				}
			}
		}
	}()

	println("ok")

}
