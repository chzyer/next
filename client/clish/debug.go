package clish

import (
	"bytes"
	"fmt"
	"runtime"

	"github.com/chzyer/flagly"
	"github.com/chzyer/readline"
	"gopkg.in/logex.v1"
)

type ShellDebug struct {
	Goroutine *ShellDebugGoroutine `flagly:"handler"`
	Log       *ShellDebugLog       `flagly:"handler"`
	Useful    *ShellDebugUseful    `flagly:"handler"`
}

type ShellDebugUseful struct{}

func (ShellDebugUseful) FlaglyHandle(rl *readline.Instance, c Client) {
	chs := c.GetDchan().GetUsefulChan()
	buf := bytes.NewBuffer(nil)
	for _, ch := range chs {
		buf.WriteString(fmt.Sprintf("%v: %v\n",
			ch.Name(), ch.GetStat().String(),
		))
	}
	fmt.Fprintf(rl, "%v", buf.String())
}

type ShellDebugGoroutine struct{}

func (ShellDebugGoroutine) FlaglyHandle(rl *readline.Instance) {
	stack := make([]byte, 1024)
	n := 0
	for {
		n = runtime.Stack(stack, true)
		if n == cap(stack) {
			stack = make([]byte, cap(stack)*2)
			continue
		}
		break
	}
	fmt.Fprintln(rl, string(stack[:n]))
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
