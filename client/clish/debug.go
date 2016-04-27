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
	Name string `type:"[0]" select:"dchan.Client,controller"`
}

type Flower interface {
	GetFlow() *flow.Flow
}

func (d *DebugFlow) FlaglyHandle(c Client, h *flagly.Handler) error {
	if d.Name == "" {
		return flagly.Error("name is required")
	}
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
	Level string `type:"[0]" select:"debug,info,warn,error"`
}

func (s ShellDebugLog) FlaglyHandle() error {
	var level int
	switch s.Level {
	case "debug":
		level = 0
	case "info":
		level = 1
	case "warn":
		level = 2
	case "error":
		level = 3
	default:
		return fmt.Errorf("current log level: %v", logex.DebugLevel)
	}

	if level == -1 {
		return fmt.Errorf("current log level: %v", logex.DebugLevel)
	}
	if level > 3 {
		return flagly.Errorf(fmt.Sprintf("invalid level: %v", level))
	}
	logex.DebugLevel = level
	return fmt.Errorf("log level set to %v", logex.DebugLevel)
}
