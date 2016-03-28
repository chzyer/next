package client

import (
	"strconv"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/controller"
	"github.com/chzyer/next/datachannel"
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

	ctl *controller.Client

	dcs   *datachannel.Client
	dcIn  chan *packet.Packet
	dcOut chan *packet.Packet
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
	port := remoteCfg.DataChannel
	session := packet.NewSessionIV(
		uint16(remoteCfg.UserId), uint16(port), []byte(remoteCfg.Token))

	dcs := datachannel.NewClient(c.flow,
		c.cfg.GetHostName(), port,
		session, c.dcIn, c.dcOut)

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
	c.dcs = dcs

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
	if c.tun != nil {
		c.tun.ConfigUpdate(a)
	}
	return nil
}

func (c *Client) initRoute() {
	c.route = route.NewRoute(c.flow, c.tun.Name())
	if err := c.route.Load(c.cfg.RouteFile); err != nil {
		logex.Error(err)
	}
}

func (c *Client) initController(toDC chan<- *packet.Packet, fromDC <-chan *packet.Packet, toTun chan<- []byte) error {
	c.ctl = controller.NewClient(c.flow, c, toDC, fromDC, toTun)
	return nil
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

	if err := c.initController(c.dcIn, c.dcOut, tunIn); err != nil {
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
				p := packet.New(data, packet.DATA)
				c.ctl.Send(p)
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

// -----------------------------------------------------------------------------
// controller
func (c *Client) OnNewDC(ports []int) {
	c.dcs.UpdateRemoteAddrs(ports)
}
