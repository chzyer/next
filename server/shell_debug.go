package server

import (
	"fmt"
	"strings"

	"github.com/chzyer/flagly"
	"github.com/chzyer/logex"
	"github.com/chzyer/next/util"
	"github.com/chzyer/readline"
)

type ShellDebug struct {
	Goroutine *ShellDebugGoroutine `flagly:"handler"`
	Log       *ShellDebugLog       `flagly:"handler"`
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
