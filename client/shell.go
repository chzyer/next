package client

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"

	"github.com/chzyer/flagly"
	"github.com/chzyer/flow"
	"github.com/chzyer/readline"
	"github.com/google/shlex"

	"gopkg.in/logex.v1"
)

type Shell struct {
	flow   *flow.Flow
	Sock   string
	ln     net.Listener
	client *Client
}

func NewShell(f *flow.Flow, cli *Client, sock string) (*Shell, error) {
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return nil, logex.Trace(err)
	}
	os.Chmod(sock, 0777)
	sh := &Shell{
		Sock:   sock,
		client: cli,
		ln:     ln,
	}
	f.ForkTo(&sh.flow, sh.Close)
	return sh, nil
}

func (s *Shell) Close() {
	s.ln.Close()
	s.flow.Close()
	os.Remove(s.Sock)
}

func (s *Shell) handleConn(conn net.Conn) {
	defer conn.Close()

	homeDir := os.Getenv("HOME")
	userAcct, _ := user.Current()
	if userAcct != nil {
		homeDir = userAcct.HomeDir
	}

	hf := filepath.Join(homeDir, ".nextcli_history")
	cfg := readline.Config{
		HistoryFile: hf,
		Prompt:      " -> ",
	}
	rl, err := readline.HandleConn(cfg, conn)
	if err != nil {
		return
	}
	defer rl.Close()

	sh := &ShellCLI{}
	fset, err := flagly.Compile("", sh)
	if err != nil {
		logex.Info(err)
		return
	}
	fset.Context(rl, s.client)

	io.WriteString(rl, "Next Client CLI\n")
	for {
		command, err := rl.Readline()
		if err != nil {
			break
		}
		args, err := shlex.Split(command)
		if err != nil {
			continue
		}

		if err := fset.Run(args); err != nil {
			fmt.Fprintln(rl.Stderr(), err)
			continue
		}
	}
}

func (s *Shell) loop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			break
		}
		go s.handleConn(conn)
	}
}

type ShellCLI struct {
	Help      *flagly.CmdHelp `flagly:"handler"`
	Ping      *ShellPing      `flagly:"handler"`
	HeartBeat *ShellHeartBeat `flagly:"handler"`
	Route     *ShellRoute     `flagly:"handler"`
}

type ShellHeartBeat struct{}

func (*ShellHeartBeat) FlaglyHandle(c *Client, rl *readline.Instance) error {
	stat := c.dataChannels.GetStats()
	fmt.Fprintln(rl, stat)
	return nil
}

type ShellPing struct{}

func (*ShellPing) FlaglyHandle(c *Client) error {
	logex.Info(c)
	return nil
}
