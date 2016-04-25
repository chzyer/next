package server

import (
	"github.com/chzyer/next/mchan"
	"github.com/chzyer/next/uc"

	"gopkg.in/logex.v1"
)

var (
	ErrWrongUserPassword = logex.Define("wrong username or password")
	ErrNotReady          = logex.Define("not ready")
)

func (h *HttpApi) Auth(req *mchan.Req) interface{} {
	var authReq *uc.AuthRequest
	if err := req.Unmarshal(&authReq); err != nil {
		return err
	}

	authInfo, err := authReq.Decode(h.key, h.clock.Unix())
	if err != nil {
		return err
	}

	u := h.users.LoginByName(authInfo.UserName, string(authInfo.Passcode))
	if u == nil {
		return ErrWrongUserPassword
	}

	if h.delegate.GetDataChannel() == -1 {
		return ErrNotReady
	}

	if u.Net == nil {
		u.Net = h.delegate.AllocIP()
	}

	logex.Info("login success, fetching datachannel")
	auth := &uc.AuthResponse{
		Gateway:     h.delegate.GetGateway().String(),
		UserId:      int(u.Id),
		INet:        u.Net.String(),
		MTU:         h.delegate.GetMTU(),
		Token:       u.Token,
		ChannelType: h.delegate.GetChannelType(),
		DataChannel: h.delegate.GetDataChannel(),
	}
	h.delegate.OnNewUser(int(u.Id))
	return auth
}

func (h *HttpApi) Time(req *mchan.Req) interface{} {
	return h.clock.Unix()
}
