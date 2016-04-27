package client

import (
	"github.com/chzyer/logex"
	"github.com/chzyer/next/dchan"
)

type DchanDelegate struct {
	c *Client
}

func (d *DchanDelegate) OnAllBackoff(cli *dchan.Client) {
	logex.Info("all dchan is backoff")
	d.c.NeedLogin()
}

func (d *DchanDelegate) OnLinkRefused() {
	d.c.ctl.RequestNewDC()
}
