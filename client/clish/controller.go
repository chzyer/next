package clish

import (
	"fmt"

	"github.com/chzyer/readline"
)

type ShellController struct {
	Stage *ShellControllerStage `flagly:"handler"`
}

type ShellControllerStage struct {
}

func (*ShellControllerStage) FlaglyHandle(c Client, rl *readline.Instance) {
	info := c.ShowControllerStage()
	fmt.Fprintf(rl, "staging: %v\n", len(info))
}
