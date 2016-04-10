package client

import (
	"fmt"

	"github.com/chzyer/flagly"
	"github.com/chzyer/next/ip"
	"github.com/chzyer/readline"
	"gopkg.in/logex.v1"
)

type ShellCLI struct {
	Help       *flagly.CmdHelp  `flagly:"handler"`
	Ping       *ShellPing       `flagly:"handler"`
	HeartBeat  *ShellHeartBeat  `flagly:"handler"`
	Route      *ShellRoute      `flagly:"handler"`
	Dig        *ShellDig        `flagly:"handler"`
	Controller *ShellController `flagly:"handler"`
	Debug      *ShellDebug      `flagly:"handler"`
}

type ShellDig struct {
	Host string `type:"[0]"`
}

func (sh *ShellDig) FlaglyDesc() string {
	return "DNS lookup utility"
}

func (sh *ShellDig) FlaglyHandle(c *Client, rl *readline.Instance) error {
	if sh.Host == "" {
		return flagly.Error("host is required")
	}
	addrs, err := ip.LookupHost(sh.Host)
	if err != nil {
		return flagly.Error(err.Error())
	}
	for _, addr := range addrs {
		fmt.Fprintln(rl, addr)
	}
	return nil
}

type ShellHeartBeat struct{}

func (ShellHeartBeat) FlaglyDesc() string {
	return "show the heartbeat stat"
}

func (*ShellHeartBeat) FlaglyHandle(c *Client, rl *readline.Instance) error {
	stat := c.dcCli.GetStats()
	fmt.Fprintln(rl, stat)
	return nil
}

type ShellPing struct{}

func (*ShellPing) FlaglyHandle(c *Client) error {
	logex.Info(c)
	return nil
}
