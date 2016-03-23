package server

import (
	"fmt"
	"io"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flagly"
	"github.com/chzyer/readline"
)

type ShellUser struct {
	Show *ShellUserShow `flaglyHandler`
	Add  *ShellUserAdd  `flaglyHandler`
}

type ShellUserAdd struct {
	Name string `[0]`
}

func (c *ShellUserAdd) FlaglyHandle(s *Server, rl *readline.Instance) error {
	if c.Name == "" {
		return flagly.Error("missing name")
	}
	u := s.uc.Find(c.Name)
	if u != nil {
		return flagly.Error(fmt.Sprintf("name '%s' already exists", c.Name))
	}
	// TODO: vps can't display "password: "
	pasw, err := rl.ReadPassword("password: ")
	if err != nil {
		return fmt.Errorf("aborted")
	}
	s.uc.Register(c.Name, string(pasw))
	err = s.uc.Save(s.cfg.DBPath)
	if err != nil {
		logex.Error(err)
	}
	// TODO: flagly can't report it
	return err
}

type ShellUserShow struct{}

func (ShellUserShow) FlaglyHandle(s *Server, rl *readline.Instance) error {
	for _, u := range s.uc.Show() {
		io.WriteString(rl, u.String()+"\n")
	}
	return nil
}
