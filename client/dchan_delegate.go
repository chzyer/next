package client

import (
	"github.com/chzyer/next/dchan"
	"gopkg.in/logex.v1"
)

type DchanDelegate struct {
	c *Client
}

func (d *DchanDelegate) OnAllBackoff(cli *dchan.Client) {
	logex.Info("all dchan is backoff")
	cli.Close()
	d.c.NeedLogin()
}

func (d *DchanDelegate) OnLinkRefused() {
	d.c.ctl.RequestNewDC()
}
