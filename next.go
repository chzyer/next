package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/chzyer/flagly"
	"github.com/chzyer/flow"
	"github.com/chzyer/next/client"
	"github.com/chzyer/next/server"
	"github.com/chzyer/next/uc"
	"github.com/chzyer/next/util"
	"github.com/chzyer/readline"
	"gopkg.in/logex.v1"
)

type Next struct {
	Server *server.Config `flagly:"handler"`
	Client *client.Config `flagly:"handler"`
	GenKey *NextGenKey    `flagly:"handler"`
	Login  *NextLogin     `flagly:"handler"`
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
	Sock string   `default:"/tmp/next.sock"`
	Args []string `type:"[]"`
}

func (n *NextShell) FlaglyHandle(f *flow.Flow) error {
	defer f.Close()

	conn, err := net.Dial("unix", n.Sock)
	if err != nil {
		return err
	}
	defer conn.Close()

	cli, err := readline.NewRemoteCli(conn)
	if err != nil {
		return err
	}
	var source io.Reader
	if len(n.Args) > 0 {
		cli.MarkIsTerminal(false)
		source = bytes.NewBufferString(strings.Join(n.Args, " ") + "\n")
	} else {
		source = os.Stdin
	}
	return cli.ServeBy(source)
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

// -----------------------------------------------------------------------------

type NextLogin struct {
	User   string
	Key    string
	Remote string `type:"[0]"`
}

func (l *NextLogin) FlaglyHandle(f *flow.Flow) error {
	defer f.Close()
	var err error

	if l.Remote == "" {
		return flagly.Error("remote host is required")
	}

	if l.Key == "" {
		return flagly.Error("key can't be empty")
	}

	if l.User == "" {
		l.User, err = readline.Line("username: ")
		if err != nil {
			return nil
		}
	}

	pswd, err := readline.Password("password: ")
	if err != nil {
		return nil
	}

	cli := client.NewHTTP(client.FixHost(l.Remote), l.User, string(pswd), []byte(l.Key))
	if err := cli.Login(func(resp *uc.AuthResponse) error {
		ret, _ := json.MarshalIndent(resp, "", "\t")
		println(string(ret))
		return nil
	}); err != nil {
		return err
	}
	return nil
}
