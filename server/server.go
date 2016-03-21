package server

import (
	"fmt"
	"strconv"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/ip"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/uc"
	"github.com/chzyer/next/util/clock"
	"gopkg.in/logex.v1"
)

type Server struct {
	cfg   *Config
	flow  *flow.Flow
	uc    *uc.Users
	cl    *clock.Clock
	shell *Shell
	dhcp  *ip.DHCP

	dataChannel *MultiDataChannel
}

func New(cfg *Config, f *flow.Flow) *Server {
	*f.Debug = cfg.DebugFlow
	svr := &Server{
		cfg:  cfg,
		flow: f,
		uc:   uc.NewUsers(),
		cl:   clock.New(),
	}
	f.SetOnClose(svr.Close)

	err := svr.uc.Load(cfg.DBPath)
	if err != nil {
		logex.Error("load user info fail:", err)
	} else {
		logex.Info("loading user info from", strconv.Quote(cfg.DBPath))
	}
	dhcp := ip.NewDHCP(cfg.Net)
	logex.Info("creating dhcp for", cfg.Net)
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
	logex.Info("listen debug sock in", strconv.Quote(s.cfg.Sock))
	shell.loop()
}

func (s *Server) runHttp() {
	api := NewHttpApi(s.cfg.HTTP, s.uc, s.cl, &HttpApiConfig{
		AesKey:   []byte(s.cfg.HTTPAes),
		CertFile: s.cfg.HTTPCert,
		KeyFile:  s.cfg.HTTPKey,
	}, s)
	logex.Info("listen HTTP Api at", s.cfg.HTTP)
	if err := api.Run(); err != nil {
		s.flow.Error(err)
	}
}

func (s *Server) loadDataChannel() {
	s.dataChannel = NewMultiDataChannel(s.flow, s)
	s.dataChannel.Start(1)
}

func (s *Server) Run() {
	go s.runHttp()
	go s.runShell()
	go s.loadDataChannel()
}

// -----------------------------------------------------------------------------
// HTTP_USER

func (s *Server) OnNewUser(userId int) {
	u := s.uc.FindId(userId)
	if u == nil {
		logex.Error("on new user but user is not exists!", userId)
		return
	}
	logex.Infof("new user is coming: Id: %v, Name: %v", u.Id, u.Name)
}

func (s *Server) GetUserChannel(id int) (in, out chan *packet.Packet, err error) {
	u := s.uc.FindId(id)
	if u == nil {
		err = uc.ErrUserNotFound.Trace()
		return
	}
	in, out = u.GetChannel()
	return
}

func (s *Server) GetUserToken(id int) string {
	u := s.uc.FindId(id)
	if u == nil {
		return ""
	}
	return u.Token
}

func (s *Server) Close() {
	if s.shell != nil {
		s.shell.Close()
	}
}

// -----------------------------------------------------------------------------
// HTTP

func (s *Server) AllocIP() *ip.IP {
	return s.dhcp.Alloc()
}

func (s *Server) GetGateway() *ip.IPNet {
	return s.dhcp.IPNet
}

func (s *Server) GetMTU() int {
	return s.cfg.MTU
}

func (s *Server) GetDataChannel(host string) string {
	return fmt.Sprintf("%s:%v", host, s.dataChannel.GetDataChannel())
}
