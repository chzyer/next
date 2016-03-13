package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/chzyer/next/uc"

	"gopkg.in/logex.v1"
)

var (
	ErrWrongUserPassword = logex.Define("wrong username or password")
)

func (h *HttpApi) Auth(w http.ResponseWriter, req *http.Request) {
	var authReq *uc.AuthRequest
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		h.replyError(w, err)
		return
	}
	if err := json.Unmarshal(body, &authReq); err != nil {
		h.replyError(w, err)
		return
	}

	authInfo, err := authReq.Decode(h.cfg.AesKey, h.clock.Unix())
	if err != nil {
		h.replyError(w, err)
		return
	}

	u := h.users.LoginByName(authInfo.UserName, string(authInfo.Passcode))
	if u == nil {
		h.replyError(w, ErrWrongUserPassword)
		return
	}
	if u.Net == nil {
		u.Net = h.svr.dhcp.Alloc()
	}
	ret := &uc.AuthResponse{
		Gateway: h.svr.dhcp.IPNet.String(),
		UserId:  int(u.Id),
		INet:    u.Net.String(),
		MTU:     h.svr.cfg.MTU,
		Token:   u.Token,
	}
	h.reply(w, ret)
}

func (h *HttpApi) Time(w http.ResponseWriter, req *http.Request) {
	h.reply(w, h.clock.Unix())
}
