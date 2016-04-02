package client

import (
	"fmt"

	"github.com/chzyer/readline"
)

type ShellController struct {
	Stage *ShellControllerStage `flagly:"handler"`
}

type ShellControllerStage struct {
}

func (*ShellControllerStage) FlaglyHandle(c *Client, rl *readline.Instance) {
	info := c.ctl.ShowStage()
	fmt.Fprintf(rl, "staging: %v\n", len(info))
}
