package tunnel

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

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
	Name          string
	fd            *os.File
	stopChan      chan struct{}
	readChan      chan []byte
	readReplyChan chan reply
}

type reply struct {
	n   int
	err error
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
		Config:        cfg,
		fd:            fd,
		Name:          fmt.Sprintf(cfg.NameLayout, cfg.DevId),
		stopChan:      make(chan struct{}),
		readChan:      make(chan []byte),
		readReplyChan: make(chan reply),
	}
	go t.loop()
	if err := syscall.SetNonblock(int(fd.Fd()), true); err != nil {
		return nil, logex.Trace(err)
	}
	if err := t.setupTun(); err != nil {
		return nil, logex.Trace(err)
	}
	return t, nil
}

var errInfo = "resource temporarily unavailable"

func (t *Instance) loop() {
main:
	for {
		select {
		case b := <-t.readChan:
			for {
				n, err := t.fd.Read(b)
				if err != nil && strings.Contains(err.Error(), errInfo) {
					select {
					case <-time.After(500 * time.Millisecond):
					case <-t.stopChan:
						return
					}
				} else {
					t.readReplyChan <- reply{n, err}
					continue main
				}
			}
		case <-t.stopChan:
			return
		}
	}
}

func (t *Instance) Read(b []byte) (int, error) {
	t.readChan <- b
	select {
	case r := <-t.readReplyChan:
		return r.n, r.err
	case <-t.stopChan:
		return 0, io.EOF
	}
}

func (t *Instance) Write(b []byte) (int, error) {
	return t.fd.Write(b)
}

func (t *Instance) Close() error {
	close(t.stopChan)
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
