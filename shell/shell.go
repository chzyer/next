package shell

import (
	"time"

	"github.com/chzyer/flow"
	"github.com/chzyer/readline"
)

type NextShell struct {
}

func (NextShell) shell(f *flow.Flow) error {
	f.Add(1)
	rl, err := readline.New("> ")
	if err != nil {
		return err
	}
	defer func() {
		f.Done()
		rl.Close()
		f.Close()
	}()

	for !f.IsClosed() {
		cmd, err := rl.Readline()
		if err != nil {
			break
		}
		time.Sleep(time.Second)
		println(cmd)
	}
	return nil
}

func (n NextShell) FlaglyHandle(f *flow.Flow) error {
	go n.shell(f)
	return nil
}

func (NextShell) FlaglyDesc() string {
	return "shell mode to configure"
}
