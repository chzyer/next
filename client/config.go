package client

import (
	"fmt"
	"strings"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
)

type Config struct {
	Debug      bool
	DebugStack bool `default:"true"`
	DebugFlow  bool

	DevId    int
	UserName string
	Password string
	AesKey   string

	Sock string `desc:"unixsock for interactive with" default:"/tmp/next.sock"`

	Host2 string `host`
	Host  string `[0]`
}

func (c *Config) FlaglyVerify() error {
	if c.Host == "" {
		c.Host = c.Host2
	}
	if c.Host == "" {
		return fmt.Errorf("host is empty")
	}
	if !strings.Contains(c.Host, ":") {
		c.Host += ":11311"
	}
	if !strings.HasPrefix(c.Host, "http") {
		c.Host = "http://" + c.Host
	}
	if c.AesKey == "" {
		return fmt.Errorf("aeskey is required")
	}
	if c.UserName == "" {
		return fmt.Errorf("username is missing")
	}
	if c.Password == "" {
		return fmt.Errorf("password is missing")
	}

	flow.DefaultDebug = c.DebugFlow
	logex.ShowCode = c.DebugStack
	return nil
}

func (c *Config) FlaglyHandle(f *flow.Flow) error {
	New(c, f).Run()
	return nil
}

func (c *Config) FlaglyDesc() string {
	return "client mode"
}
