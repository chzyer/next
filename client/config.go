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
	AesKey    string `name:"key"`
	RouteFile string `default:"routes.conf"`
	Pprof     string `default:":10060"`

	Sock string `desc:"unixsock for interactive with" default:"/tmp/next.sock"`

	Host2 string `name:"host"`
	Host  string `type:"[0]"`
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

func FixHost(host string) string {
	if !strings.Contains(host, ":") {
		host += ":11311"
	}
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}
	return host
}

func (c *Config) FlaglyVerify() error {
	if c.Host == "" {
		c.Host = c.Host2
	}
	if c.Host == "" {
		return fmt.Errorf("host is empty")
	}
	c.Host = FixHost(c.Host)

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
