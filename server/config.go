package server

import (
	"errors"

	"github.com/chzyer/flagly"
	"github.com/chzyer/flow"
)

type Config struct {
	Debug bool `desc:"turn on debug"`

	HTTP     string `desc:"listen http port" default:":11311"`
	HTTPAes  string `desc:"http aes key; required"`
	HTTPCert string `desc:"https cert file path"`
	HTTPKey  string `desc:"https key file path"`

	DBPath string `desc:"filepath to persist user info"`
}

func (c *Config) FlaglyVerify() error {
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
