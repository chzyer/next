package server

import (
	"github.com/chzyer/flow"
	"github.com/chzyer/next/ip"
	"github.com/chzyer/next/mchan"
	"github.com/chzyer/next/uc"
	"github.com/chzyer/next/util/clock"
)

type HttpApi struct {
	listen   string
	clock    *clock.Clock
	key      []byte
	users    *uc.Users
	server   *mchan.Server
	delegate HttpDelegate
}

type HttpDelegate interface {
	AllocIP() *ip.IP
	GetGateway() *ip.IPNet
	GetMTU() int
	GetDataChannel() int
	OnNewUser(userId int)
}

func NewHttpApi(f *flow.Flow, listen string, users *uc.Users, ct *clock.Clock, key []byte, cfg *mchan.SvrConf, delegate HttpDelegate) *HttpApi {
	return &HttpApi{
		clock:    ct,
		key:      key,
		users:    users,
		server:   mchan.NewServer(f, listen, ct, key, cfg),
		delegate: delegate,
	}
}

func (h *HttpApi) Run() error {
	h.server.HandleFunc("/auth", h.Auth)
	h.server.HandleFunc("/time", h.Time)
	return h.server.Run()
}
