package server

import (
	"errors"

	"github.com/chzyer/flow"
)

type Config struct {
	Debug bool `desc:"turn on debug"`

	HTTP     string `desc:"listen http port" default:":11311"`
	HTTPAes  string `desc:"http aes key; required"`
	HTTPCert string `desc:"https cert file path"`
	HTTPKey  string `desc:"https key file path"`
}

func (c *Config) FlaglyVerify() error {
	if c.HTTPAes == "" {
		return errors.New("httpaes is required, please try `next genkey` to genreate one")
	}
	return nil
}

func (c *Config) FlaglyHandle(f *flow.Flow) error {
	New(c, f).Run()
	return nil
}

func (c *Config) FlaglyDesc() string {
	return "server mode"
}
