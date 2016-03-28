package client

import (
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
)

type Config struct {
	Debug      bool
	DebugStack bool `default:"true"`
	DebugFlow  bool

	DevId     int
	UserName  string
	Password  string
	AesKey    string
	RouteFile string `default:"routes.conf"`

	Sock string `desc:"unixsock for interactive with" default:"/tmp/next.sock"`

	Host2 string `name:"host"`
	Host  string `name:"[0]"`
}

func (c *Config) GetHostName() string {
	u, err := url.Parse(c.Host)
	if err != nil {
		panic(err)
	}
	idx := strings.Index(u.Host, ":")
	if idx > 0 {
		u.Host = u.Host[:idx]
	}
	return u.Host
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
