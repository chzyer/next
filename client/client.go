package client

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/controller"
	"github.com/chzyer/next/dchan"
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
	HTTP  *HTTP

	ctl *controller.Client

	dcCli *dchan.Client
	dcIn  chan *packet.Packet
	dcOut chan *packet.Packet

	needLoginChan chan struct{}
}

func New(cfg *Config, f *flow.Flow) *Client {
	cli := &Client{
		cfg:           cfg,
		flow:          f,
		dcIn:          make(chan *packet.Packet),
		dcOut:         make(chan *packet.Packet),
		HTTP:          NewHTTP(cfg.Host, cfg.UserName, cfg.Password, []byte(cfg.AesKey)),
		needLoginChan: make(chan struct{}, 1),
	}
	http.DefaultClient.Timeout = 10 * time.Second
	return cli
}

func (c *Client) Close() {
	c.flow.Close()
}

func (c *Client) reloginLoop() {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()

loop:
	for {
		select {
		case <-c.needLoginChan:
			logex.Info("need to login")
		resend:
			if err := c.HTTP.Login(c.onLogin); err != nil {
				logex.Error(err)
				switch c.flow.CloseOrWait(time.Second) {
				case flow.F_TIMEOUT:
					goto resend
				case flow.F_CLOSED:
					break loop
				}
			}
		case <-c.flow.IsClose():
			break loop
		}
	}
}

func (c *Client) initDataChannel(remoteCfg *uc.AuthResponse) (err error) {
	port := remoteCfg.DataChannel
	session := packet.NewSessionIV(
		uint16(remoteCfg.UserId), uint16(port), []byte(remoteCfg.Token))

	if c.dcCli != nil {
		c.dcCli.Close()
		c.dcCli = nil
	}

	dcCli := dchan.NewClient(c.flow, session, c.dcIn, c.dcOut)
	dcCli.AddHost(c.cfg.GetHostName(), port)
	dcCli.SetOnAllBackoff(func() {
		logex.Info("all dchan is backoff")
		dcCli.Close()
		logex.Info("send needLogin chan")
		select {
		case c.needLoginChan <- struct{}{}:
			logex.Info("send needLogin chan success")
		case <-c.flow.IsClose():
		}
	})
	c.dcCli = dcCli
	dcCli.Run()
	logex.Info("datachannel inited:", dcCli.Ports())
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

func (c *Client) onLogin(remoteCfg *uc.AuthResponse) error {
	logex.Pretty(remoteCfg)

	err := c.initDataChannel(remoteCfg)
	if err != nil {
		return logex.Trace(err)
	}

	if c.tun != nil {
		c.tun.ConfigUpdate(remoteCfg)
		return nil
	}

	tunIn, tunOut, err := c.initTun(remoteCfg)
	if err != nil {
		return logex.Trace(err)
	}

	if err := c.initController(c.dcIn, c.dcOut, tunIn); err != nil {
		return logex.Trace(err)
	}

	c.initRouteTable()

	go c.tunToControllerLoop(tunOut)

	return nil
}

func (c *Client) tunToControllerLoop(tunOut <-chan []byte) {
	c.flow.Add(1)
	defer c.flow.DoneAndClose()
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
}

func (c *Client) initRouteTable() {
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
	if err := c.runShell(); err != nil {
		c.flow.Error(err)
		return
	}

relogin:
	if err := c.HTTP.Login(c.onLogin); err != nil {
		if strings.Contains(err.Error(), "timeout") {
			logex.Info("try to relogin")
			goto relogin
		}
		c.flow.Error(err)
		return
	}

	go c.reloginLoop()
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
	c.dcCli.UpdateRemoteAddrs(c.cfg.GetHostName(), ports)
}

func (c *Client) SaveRoute() error {
	return c.route.Save(c.cfg.RouteFile)
}
