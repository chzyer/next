package client

import (
	"fmt"
	"strings"

	"github.com/chzyer/flow"
)

type Config struct {
	Debug    bool
	DevId    int
	UserName string
	Password string
	AesKey   string

	RemoteHost string `[0]`
}

func (c *Config) FlaglyVerify() error {
	if c.RemoteHost == "" {
		return fmt.Errorf("remoteHost is empty")
	}
	if !strings.Contains(c.RemoteHost, ":") {
		c.RemoteHost += ":11311"
	}
	if !strings.HasPrefix(c.RemoteHost, "http") {
		c.RemoteHost = "http://" + c.RemoteHost
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
	return nil
}

func (c *Config) FlaglyHandle(f *flow.Flow) error {
	New(c, f).Run()
	return nil
}

func (c *Config) FlaglyDesc() string {
	return "client mode"
}
