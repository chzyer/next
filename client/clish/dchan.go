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
	Speed  *DchanSpeed  `flagly:"handler"`
}

type DchanSpeed struct{}

func (DchanSpeed) FlaglyHandle(c Client) error {
	ch, err := c.GetDchan()
	if err != nil {
		return err
	}
	info := ch.GetSpeedInfo()
	return fmt.Errorf("upload:   %v/s\ndownload: %v/s", info.Upload, info.Download)
}

type DchanUseful struct{}

func (DchanUseful) FlaglyHandle(c Client) error {
	ch, err := c.GetDchan()
	if err != nil {
		return err
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
	Name string `type:"[0]"`
}

func (d *DchanClose) FlaglyHandle(c Client) error {
	if d.Name == "" {
		return flagly.Error("name is both required")
	}
	ch, err := c.GetDchan()
	if err != nil {
		return err
	}
	return ch.CloseChannel(d.Name)
}

type DchanList struct{}

func (DchanList) FlaglyHandle(c Client) error {
	stat, err := c.GetDataChannelStat()
	if err != nil {
		return err
	}

	return fmt.Errorf("%v", strings.TrimSpace(stat))
}
