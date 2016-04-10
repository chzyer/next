package client

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"

	"github.com/chzyer/flagly"
	"github.com/chzyer/flow"
	"github.com/chzyer/next/client/clish"
	"github.com/chzyer/next/controller"
	"github.com/chzyer/next/dchan"
	"github.com/chzyer/next/route"
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

	sh := &clish.CLI{}
	fset, err := flagly.Compile("", sh)
	if err != nil {
		logex.Info(err)
		return
	}

	homeDir := os.Getenv("HOME")
	userAcct, _ := user.Current()
	if userAcct != nil {
		homeDir = userAcct.HomeDir
	}

	hf := filepath.Join(homeDir, ".nextcli_history")
	cfg := readline.Config{
		HistoryFile:  hf,
		Prompt:       " -> ",
		AutoComplete: &readline.SegmentComplete{fset.Completer()},
	}
	rl, err := readline.HandleConn(cfg, conn)
	if err != nil {
		return
	}
	defer rl.Close()

	var client clish.Client = s.client
	fset.Context(rl, &client)

	if rl.Config.FuncIsTerminal() {
		fmt.Fprintln(rl, "Next Client CLI")
	}
	for {
		command, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(command) == 0 {
				break
			} else {
				continue
			}
		} else if err != nil {
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

// Shell Delegate
// -----------------------------------------------------------------------------

func (c *Client) GetDataChannelStat() string {
	return c.dcCli.GetStats()
}

func (c *Client) ShowControllerStage() []controller.StageInfo {
	return c.ctl.ShowStage()
}

func (c *Client) GetController() *controller.Client {
	return c.ctl
}

func (c *Client) GetDchan() *dchan.Client {
	return c.dcCli
}

func (c *Client) GetRoute() *route.Route {
	return c.route
}
