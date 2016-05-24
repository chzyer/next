package client

import "github.com/chzyer/next/dchan"

type DchanDelegate struct {
	client *Client
}

func (d *DchanDelegate) OnAllBackoff(cli *dchan.Client) {
	d.client.OnAllBackoff()
}

func (d *DchanDelegate) OnLinkRefused() {
	d.client.ctl.RequestNewDC()
}
