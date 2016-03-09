package server

import (
	"github.com/chzyer/flow"
	"github.com/chzyer/next/uc"
	"github.com/chzyer/next/util/clock"
)

type Server struct {
	cfg  *Config
	flow *flow.Flow
	uc   *uc.Users
	cl   *clock.Clock
}

func New(cfg *Config, f *flow.Flow) *Server {
	svr := &Server{
		cfg:  cfg,
		flow: f,
		uc:   uc.NewUsers(),
		cl:   clock.New(),
	}
	return svr
}

func (s *Server) runHttp() {
	api := NewHttpApi(s.cfg.HTTP, s.uc, s.cl, &HttpApiConfig{
		AesKey:   []byte(s.cfg.HTTPAes),
		CertFile: s.cfg.HTTPCert,
		KeyFile:  s.cfg.HTTPKey,
	})
	if err := api.Run(); err != nil {
		s.flow.Error(err)
	}
}

func (s *Server) Run() {
	go s.runHttp()
}
