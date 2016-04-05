package server

import (
	"encoding/json"

	"github.com/chzyer/next/uc"

	"gopkg.in/logex.v1"
)

var (
	ErrWrongUserPassword = logex.Define("wrong username or password")
)

func (h *HttpApi) Auth(pl []byte) interface{} {
	var authReq *uc.AuthRequest
	if err := json.Unmarshal(pl, &authReq); err != nil {
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
	if u.Net == nil {
		u.Net = h.delegate.AllocIP()
	}

	h.delegate.OnNewUser(int(u.Id))

	return &uc.AuthResponse{
		Gateway:     h.delegate.GetGateway().String(),
		UserId:      int(u.Id),
		INet:        u.Net.String(),
		MTU:         h.delegate.GetMTU(),
		Token:       u.Token,
		DataChannel: h.delegate.GetDataChannel(),
	}
}

func (h *HttpApi) Time(pl []byte) interface{} {
	return h.clock.Unix()
}
