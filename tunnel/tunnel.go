package tunnel

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"

	"gopkg.in/logex.v1"
)

type Config struct {
	DevId      int
	Gateway    *net.IPNet
	MTU        int
	Debug      bool
	NameLayout string
}

type Instance struct {
	*Config
	Name string
	fd   *os.File
}

func New(cfg *Config) (*Instance, error) {
	fd, err := OpenTun(cfg.DevId)
	if err != nil {
		return nil, logex.Trace(err)
	}
	if cfg.NameLayout == "" {
		cfg.NameLayout = "utun%d"
	}
	t := &Instance{
		Config: cfg,
		fd:     fd,
		Name:   fmt.Sprintf(cfg.NameLayout, cfg.DevId),
	}
	if err := t.setupTun(); err != nil {
		return nil, logex.Trace(err)
	}
	return t, nil
}

func (t *Instance) Read(b []byte) (int, error) {
	return t.fd.Read(b)
}

func (t *Instance) Write(b []byte) (int, error) {
	return t.fd.Write(b)
}

func (t *Instance) Close() error {
	return t.fd.Close()
}

func (t *Instance) shell(s string) error {
	if t.Debug {
		logex.Info(s)
	}
	cmd := exec.Command("/bin/bash", "-c", s)
	ret, err := cmd.CombinedOutput()
	if t.Debug && len(ret) > 0 {
		logex.Info(string(ret))
	}
	if err == nil {
		return nil
	}
	return errors.New(s + ": " + string(ret))
}
