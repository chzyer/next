package client

import (
	"strconv"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/route"
	"github.com/chzyer/next/uc"
	"github.com/chzyer/next/util/clock"
	"gopkg.in/logex.v1"
)

type Client struct {
	cfg   *Config
	clock *clock.Clock
	flow  *flow.Flow
	tun   *Tun
	shell *Shell
	route *route.Route

	dataChannels *DataChannels
	dcIn         chan *packet.Packet
	dcOut        chan *packet.Packet
}

func New(cfg *Config, f *flow.Flow) *Client {
	cli := &Client{
		cfg:   cfg,
		flow:  f,
		dcIn:  make(chan *packet.Packet),
		dcOut: make(chan *packet.Packet),
	}
	return cli
}

func (c *Client) Close() {
	c.flow.Close()
}

func (c *Client) initDataChannel(remoteCfg *uc.AuthResponse) (err error) {
	port := remoteCfg.GetDataChannelPort()
	session := packet.NewSessionIV(
		uint16(remoteCfg.UserId), uint16(port), []byte(remoteCfg.Token))

	dcs := NewDataChannels(c.flow, []string{remoteCfg.DataChannel}, session,
		c.dcIn, c.dcOut)
	dcs.SetOnAllChannelsBackoff(func() {
		dcs.Close()
		for {
			resp, err := c.Login()
			if err != nil {
				logex.Error(err)
				time.Sleep(2 * time.Second)
				continue
			}
			*remoteCfg = *resp
			break
		}
	})
	c.dataChannels = dcs

	return nil
}

func (c *Client) initTun(remoteCfg *uc.AuthResponse) (in, out chan []byte, err error) {
	in = make(chan []byte)
	out = make(chan []byte)
	tun, err := newTun(c.flow, remoteCfg, c.cfg)
	if err != nil {
		return nil, nil, err
	}
	tun.Run(in, out)
	c.tun = tun
	return in, out, nil
}

func (c *Client) onLogin(a *uc.AuthResponse) error {
	err := c.initDataChannel(a)
	if err != nil {
		return logex.Trace(err)
	}
	return nil
}

func (c *Client) initRoute() {
	c.route = route.NewRoute(c.flow, c.tun.Name())
	// c.route.Load(fp)
}

func (c *Client) Run() {
	remoteCfg, err := c.Login()
	if err != nil {
		c.flow.Error(err)
		return
	}

	tunIn, tunOut, err := c.initTun(remoteCfg)
	if err != nil {
		c.flow.Error(err)
		return
	}

	c.initRoute()

	if err := c.runShell(); err != nil {
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
				c.dcIn <- packet.New(data, packet.Data)
			case pRecv := <-c.dcOut:
				if pRecv.Type == packet.Data {
					tunIn <- pRecv.Data()
				}
			}
		}
	}()

	println("ok")

}

func (c *Client) runShell() error {
	shell, err := NewShell(c.flow, c, c.cfg.Sock)
	if err != nil {
		return err
	}
	c.shell = shell
	logex.Info("listen debug sock in", strconv.Quote(c.cfg.Sock))
	go shell.loop()
	return nil
}
