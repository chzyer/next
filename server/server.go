package server

import (
	"net/http"
	"strconv"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/controller"
	"github.com/chzyer/next/dchan"
	"github.com/chzyer/next/ip"
	"github.com/chzyer/next/mchan"
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
	tun   *Tun

	controllerGroup *controller.Group
	dchanServer     *dchan.Server
	dchanGroup      *dchan.ListenerGroup
}

func New(cfg *Config, f *flow.Flow) *Server {
	svr := &Server{
		cfg:  cfg,
		flow: f,
		uc:   uc.NewUsers(),
		cl:   clock.New(),
	}
	f.SetOnClose(svr.Close)
	svr.dchanServer = dchan.NewServer(svr.flow, svr)

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
	api := NewHttpApi(s.flow, s.cfg.HTTP, s.uc, s.cl, []byte(s.cfg.HTTPAes), &mchan.SvrConf{
		CertFile: s.cfg.HTTPCert,
		KeyFile:  s.cfg.HTTPKey,
	}, s)
	logex.Info("listen HTTP Api at", s.cfg.HTTP)
	if err := api.Run(); err != nil {
		s.flow.Error(err)
	}
}

func (s *Server) loadDataChannel() {
	s.dchanGroup = dchan.NewListenerGroup(s.flow, s)
	go s.dchanGroup.Run(4)
}

func (s *Server) initAndRunTun() error {
	tun, err := newTun(s.flow, s.cfg)
	if err != nil {
		return err
	}
	tun.Run()
	s.tun = tun
	return nil
}

func (s *Server) initControllerGroup() {
	s.controllerGroup = controller.NewGroup(s.flow, s, s.uc, s.tun.WriteChan())
	go s.controllerGroup.RunDeliver(s.tun.ReadChan())
}

func (s *Server) runPprof() {
	err := http.ListenAndServe("localhost:6060", nil)
	if err != nil {
		s.flow.Error(err)
	}
}

func (s *Server) Run() {
	if err := s.initAndRunTun(); err != nil {
		s.flow.Error(err)
		return
	}
	s.initControllerGroup() // after tun
	go s.runPprof()
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
	logex.Debug("notify controller new user is logined")
	s.controllerGroup.UserLogin(u)

	logex.Infof("new user is coming: Id: %v, Name: %v", u.Id, u.Name)
}

// controller -> user -> datachannel
func (s *Server) GetUserChannelFromDataChannel(id int) (
	fromUser <-chan *packet.Packet, toUser chan<- *packet.Packet, err error) {
	u := s.uc.FindId(id)
	if u == nil {
		err = uc.ErrUserNotFound.Trace()
		return
	}
	fromUser, toUser = u.GetFromDataChannel()
	return
}

func (s *Server) GetUserToken(id int) ([]byte, error) {
	u := s.uc.FindId(id)
	if u == nil {
		return nil, uc.ErrUserNotFound.Trace()
	}
	return []byte(u.Token), nil
}

func (s *Server) OnDChanUpdate(port []int) {
	s.controllerGroup.OnDchanPortUpdate(port)
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

func (s *Server) GetDataChannel() int {
	return s.dchanGroup.GetDataChannel()
}

// -----------------------------------------------------------------------------
// controller

func (s *Server) GetAllDataChannel() []int {
	return s.dchanGroup.GetAllDataChannel()
}

func (s *Server) OnNewChannel(ch dchan.Channel) {
	s.dchanServer.AddChannel(ch)
}
