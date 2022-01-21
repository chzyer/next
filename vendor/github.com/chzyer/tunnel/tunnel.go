package tunnel

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync/atomic"
)

type Config struct {
	DevId      int
	Gateway    net.IP
	Mask       net.IPMask
	MTU        int
	Debug      bool
	NameLayout string
}

type Instance struct {
	*Config
	Name   string
	fd     *os.File
	CIDR   *net.IPNet
	closed int32
}

func New(cfg *Config) (*Instance, error) {
	fd, err := OpenTun(cfg.DevId)
	if err != nil {
		return nil, err
	}
	if cfg.NameLayout == "" {
		cfg.NameLayout = "utun%d"
	}
	if cfg.MTU == 0 {
		cfg.MTU = 1500
	}
	_, ipnet, err := net.ParseCIDR((&net.IPNet{cfg.Gateway, cfg.Mask}).String())
	if err != nil {
		return nil, err
	}
	t := &Instance{
		Config: cfg,
		fd:     fd,
		CIDR:   ipnet,
		Name:   fmt.Sprintf(cfg.NameLayout, cfg.DevId),
	}

	if err := t.setupTun(); err != nil {
		return nil, err
	}
	return t, nil
}

// nonthread-safe
func (t *Instance) Read(b []byte) (int, error) {
	n, err := t.fd.Read(b)
	if atomic.LoadInt32(&t.closed) == 1 {
		return 0, io.EOF
	}
	return n, err
}

func (t *Instance) Write(b []byte) (int, error) {
	return t.fd.Write(b)
}

func (t *Instance) Close() error {
	if !atomic.CompareAndSwapInt32(&t.closed, 0, 1) {
		return nil
	}
	t.close()
	return nil
}

func (t *Instance) shell(s string) error {
	cmd := exec.Command("/usr/bin/env", "bash", "-c", s)
	ret, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	return errors.New(s + ": " + string(ret))
}
