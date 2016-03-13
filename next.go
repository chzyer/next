package main

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/chzyer/flagly"
	"github.com/chzyer/flow"
	"github.com/chzyer/next/client"
	"github.com/chzyer/next/server"
	"github.com/chzyer/readline"
	"gopkg.in/logex.v1"
)

type Next struct {
	Server *server.Config `flaglyHandler`
	Client *client.Config `flaglyHandler`
	GenKey *NextGenKey    `flaglyHandler`
	Shell  *NextShell     `flaglyHandler`
}

func main() {
	fset := flagly.New(os.Args[0])
	f := flow.New(0)
	fset.Context(f)
	if err := fset.Compile(&Next{}); err != nil {
		logex.Fatal(err)
	}

	if err := fset.Run(os.Args[1:]); err != nil {
		flagly.Exit(err)
	}

	err := f.Wait()
	fset.Close()
	if err != nil {
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
	return "shell mode to configure"
}
