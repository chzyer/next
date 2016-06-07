package client_simple

import (
	"net"

	"github.com/chzyer/flow"
	"github.com/chzyer/tunnel"
)

type Config struct {
}

func (c *Config) FlaglyHandle(f *flow.Flow) error {
	tun, err := tunnel.New(&tunnel.Config{
		DevId:   12,
		Gateway: net.ParseIP("10.222.0.2"),
		Mask:    net.CIDRMask(24, 32),
		MTU:     1500,
	})
	if err != nil {
		return err
	}
	_ = tun
	return nil
}
