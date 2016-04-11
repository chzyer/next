package clish

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/chzyer/flagly"
)

type Dchan struct {
	Useful *DchanUseful `flagly:"handler"`
	Close  *DchanClose  `flagly:"handler"`
	List   *DchanList   `flagly:"handler"`
}

type DchanUseful struct{}

func (DchanUseful) FlaglyHandle(c Client) error {
	ch, err := c.GetDchan()
	if err != nil {
		return fmt.Errorf("not ready")
	}
	chs := ch.GetUsefulChan()
	buf := bytes.NewBuffer(nil)
	for _, ch := range chs {
		buf.WriteString(fmt.Sprintf("%v: %v\n",
			ch.Name(), ch.GetStat().String(),
		))
	}
	return fmt.Errorf("%v", strings.TrimSpace(buf.String()))
}

type DchanClose struct {
	Src string `type:"[0]"`
	Dst string `type:"[1]"`
}

func (d *DchanClose) FlaglyHandle(c Client) error {
	if d.Src == "" || d.Dst == "" {
		return flagly.Error("src/dst is both required")
	}
	ch, err := c.GetDchan()
	if err != nil {
		return err
	}
	return ch.CloseChannel(d.Src, d.Dst)
}

type DchanList struct{}

func (DchanList) FlaglyHandle(c Client) error {
	stat, err := c.GetDataChannelStat()
	if err != nil {
		return err
	}

	return fmt.Errorf("%v", strings.TrimSpace(stat))
}
