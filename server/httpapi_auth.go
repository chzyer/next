package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

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
		u.Net = h.delegate.AllocIP()
	}

	host := req.Host
	if idx := strings.Index(host, ":"); idx > 0 {
		host = host[:idx]
	}
	h.delegate.OnNewUser(int(u.Id))

	ret := &uc.AuthResponse{
		Gateway:     h.delegate.GetGateway().String(),
		UserId:      int(u.Id),
		INet:        u.Net.String(),
		MTU:         h.delegate.GetMTU(),
		Token:       u.Token,
		DataChannel: h.delegate.GetDataChannel(),
	}
	h.reply(w, ret)
}

func (h *HttpApi) Time(w http.ResponseWriter, req *http.Request) {
	h.reply(w, h.clock.Unix())
}
