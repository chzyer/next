package clish

import (
	"fmt"
	"strings"

	"github.com/chzyer/flagly"
	"github.com/chzyer/flow"
	"github.com/chzyer/logex"
	"github.com/chzyer/next/util"
	"github.com/chzyer/readline"
)

type ShellDebug struct {
	Goroutine *ShellDebugGoroutine `flagly:"handler"`
	Log       *ShellDebugLog       `flagly:"handler"`
	Login     *Login               `flagly:"handler"`
	Flow      *DebugFlow           `flagly:"handler"`
}

type DebugFlow struct {
	Name string `type:"[0]"`
}

type Flower interface {
	GetFlow() *flow.Flow
}

func (d *DebugFlow) FlaglyHandle(c Client) error {
	var flower Flower
	switch d.Name {
	case "dchan.Client":
		dchan, err := c.GetDchan()
		if err != nil {
			return err
		}
		flower = dchan
	case "controller":
		ctl, err := c.GetController()
		if err != nil {
			return err
		}
		flower = ctl
	}
	if flower == nil {
		return fmt.Errorf("%v is not found", d.Name)
	}
	return fmt.Errorf(string(flower.GetFlow().GetDebug()))
}

type Login struct{}

func (Login) FlaglyHandle(c Client) {
	c.Relogin()
}

type ShellDebugGoroutine struct {
	Find string `type:"[0]"`
}

func (s ShellDebugGoroutine) FlaglyHandle(rl *readline.Instance) error {
	var ret string
	if s.Find == "" {
		ret = string(util.GetRuntimeStackInfo())
	} else {
		sp := util.FindRuntimeStack(s.Find)
		ret = strings.Join(sp, "\n\n")
	}
	return fmt.Errorf(ret)
}

type ShellDebugLog struct {
	Level int `default:"-1" desc:"0: Debug, 1: Info, 2: Warn, 3: Error"`
}

func (s ShellDebugLog) FlaglyHandle(rl *readline.Instance) error {
	if s.Level == -1 {
		fmt.Fprintln(rl, "current log level:", logex.DebugLevel)
		return nil
	}
	if s.Level > 3 {
		return flagly.Error(fmt.Sprintf("invalid level: %v", s.Level))
	}
	logex.DebugLevel = s.Level
	fmt.Fprintln(rl, "log level set to", logex.DebugLevel)
	return nil
}
