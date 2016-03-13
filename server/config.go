package server

import (
	"errors"

	"github.com/chzyer/flagly"
	"github.com/chzyer/flow"
	"github.com/chzyer/next/ip"
)

func init() {
	flagly.RegisterAll(ip.IPNet{})
}

type Config struct {
	Debug bool `desc:"turn on debug"`

	HTTP     string    `desc:"listen http port" default:":11311"`
	HTTPAes  string    `desc:"http aes key; required"`
	HTTPCert string    `desc:"https cert file path"`
	HTTPKey  string    `desc:"https key file path"`
	Sock     string    `desc:"unixsock for interactive with" default:"/tmp/next.sock"`
	MTU      int       `default:"1500"`
	Net      *ip.IPNet `default:"10.8.0.1/24"`

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
	return nil
}

func (c *Config) FlaglyHandle(f *flow.Flow, h *flagly.Handler) error {
	srv := New(c, f)
	srv.Run()
	h.SetOnExit(srv.Close)
	return nil
}

func (c *Config) FlaglyDesc() string {
	return "server mode"
}
