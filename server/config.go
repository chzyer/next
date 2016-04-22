package server

import (
	"errors"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flagly"
	"github.com/chzyer/flow"
	"github.com/chzyer/next/ip"
)

func init() {
	flagly.RegisterAll(ip.IPNet{})
}

type Config struct {
	Debug      bool `desc:"turn on debug"`
	DebugStack bool `default:"true"`
	DebugFlow  bool
	DebugTun   bool

	HTTP     string    `desc:"listen http port" default:":11311"`
	HTTPAes  string    `desc:"http aes key; required"`
	HTTPCert string    `desc:"https cert file path"`
	HTTPKey  string    `desc:"https key file path"`
	Sock     string    `desc:"unixsock for interactive with" default:"/tmp/next.sock"`
	MTU      int       `default:"1500"`
	Net      *ip.IPNet `default:"10.8.0.1/24"`
	Pprof    string    `default:":10060"`
	DevId    int

	DBPath string `desc:"filepath to persist user info" default:"nextuser"`
}

func (c *Config) FlaglyVerify() error {
	if c.Net == nil {
		return errors.New("net is empty")
	}
	if c.HTTPAes == "" {
		return errors.New("httpaes is required, please try `next genkey` to genreate one")
	}
	if c.DBPath == "" {
		return errors.New("dbpath is empty")
	}

	flow.DefaultDebug = c.DebugFlow
	logex.ShowCode = c.DebugStack
	return nil
}

func (c *Config) FlaglyHandle(f *flow.Flow, h *flagly.Handler) error {
	srv := New(c, f)
	srv.Run()
	return nil
}

func (c *Config) FlaglyDesc() string {
	return "server mode"
}
