package server

import (
	"github.com/chzyer/flow"
	"github.com/chzyer/next/ip"
	"github.com/chzyer/next/uc"
	"github.com/chzyer/next/util/clock"
)

type Server struct {
	cfg   *Config
	flow  *flow.Flow
	uc    *uc.Users
	cl    *clock.Clock
	shell *Shell
	dhcp  *ip.DHCP
}

func New(cfg *Config, f *flow.Flow) *Server {
	svr := &Server{
		cfg:  cfg,
		flow: f,
		uc:   uc.NewUsers(),
		cl:   clock.New(),
	}
	svr.uc.Load(cfg.DBPath)
	dhcp := ip.NewDHCP(cfg.Net)
	svr.dhcp = dhcp

	return svr
}

func (s *Server) runShell() {
	shell, err := NewShell(s, s.cfg.Sock)
	if err != nil {
		s.flow.Error(err)
		return
	}
	s.shell = shell
	shell.loop()
}

func (s *Server) runHttp() {
	api := NewHttpApi(s.cfg.HTTP, s.uc, s.cl, &HttpApiConfig{
		AesKey:   []byte(s.cfg.HTTPAes),
		CertFile: s.cfg.HTTPCert,
		KeyFile:  s.cfg.HTTPKey,
	}, s)
	if err := api.Run(); err != nil {
		s.flow.Error(err)
	}
}

func (s *Server) Run() {
	go s.runHttp()
	go s.runShell()
}

func (s *Server) Close() {
	if s.shell != nil {
		s.shell.Close()
	}
}
