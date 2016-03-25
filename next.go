package main

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/chzyer/flagly"
	"github.com/chzyer/flow"
	"github.com/chzyer/next/client"
	"github.com/chzyer/next/server"
	"github.com/chzyer/next/util"
	"github.com/chzyer/readline"
	"gopkg.in/logex.v1"
)

type Next struct {
	Server *server.Config `flagly:"handler"`
	Client *client.Config `flagly:"handler"`
	GenKey *NextGenKey    `flagly:"handler"`
	SysEnv *SysEnv        `flagly:"handler"`
	Shell  *NextShell     `flagly:"handler"`
}

func main() {
	f := flow.New()
	fset, err := flagly.Compile(os.Args[0], &Next{})
	if err != nil {
		logex.Fatal(err)
	}
	fset.Context(f)

	if err := fset.Run(os.Args[1:]); err != nil {
		flagly.Exit(err)
	}

	if err := f.Wait(); err != nil {
		logex.Error(err)
	}
}

// -----------------------------------------------------------------------------

type NextGenKey struct{}

func (NextGenKey) FlaglyHandle(f *flow.Flow) error {
	key := make([]byte, 32)
	rand.Read(key)
	fmt.Println(fmt.Sprintf("%x", key)[:32])
	f.Close()
	return nil
}

func (NextGenKey) FlaglyDesc() string {
	return "random generate aes key"
}

// -----------------------------------------------------------------------------

type NextShell struct {
	Sock string `default:"/tmp/next.sock"`
}

func (n *NextShell) FlaglyHandle(f *flow.Flow) error {
	defer f.Close()
	return readline.DialRemote("unix", n.Sock)
}

func (NextShell) FlaglyDesc() string {
	return "shell mode"
}

// -----------------------------------------------------------------------------

type SysEnv struct {
	Iface string `default:"eth0"`
}

func (s *SysEnv) FlaglyHandle(f *flow.Flow) error {
	defer f.Close()
	sh := []string{
		"sysctl -w net.ipv4.ip_forward=1",
		"iptables --table nat --append POSTROUTING " +
			"--out-interface " + s.Iface + " --jump MASQUERADE",
	}
	for _, s := range sh {
		println(s)
		if err := util.Shell(s); err != nil {
			logex.Error(err)
		}
	}
	return nil
}

func (s *SysEnv) FlaglyDesc() string {
	return "enable ipforward and NAT, for linux"
}
