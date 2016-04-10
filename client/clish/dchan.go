package clish

import (
	"fmt"
	"strings"

	"github.com/chzyer/flagly"
)

type Dchan struct {
	Close *DchanClose `flagly:"handler"`
	List  *DchanList  `flagly:"handler"`
}

type DchanClose struct {
	Src string `type:"[0]"`
	Dst string `type:"[1]"`
}

func (d *DchanClose) FlaglyHandle(c Client) error {
	if d.Src == "" || d.Dst == "" {
		return flagly.Error("src/dst is both required")
	}
	return c.GetDchan().CloseChannel(d.Src, d.Dst)
}

type DchanList struct{}

func (DchanList) FlaglyHandle(c Client) error {
	stat := c.GetDataChannelStat()
	return fmt.Errorf("%v", strings.TrimSpace(stat))
}
