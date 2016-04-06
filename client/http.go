package client

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"gopkg.in/logex.v1"

	"github.com/chzyer/next/mchan"
	"github.com/chzyer/next/uc"
	"github.com/chzyer/next/util/clock"
)

type HTTP struct {
	Host   string
	User   string
	Pswd   string
	AesKey []byte
	clock  *clock.Clock
}

func NewHTTP(host, user, pswd string, aeskey []byte) *HTTP {
	return &HTTP{
		Host:   host,
		User:   user,
		Pswd:   pswd,
		AesKey: aeskey,
	}
}

func (h *HTTP) httpReq(ret interface{}, path string, data interface{}) error {
	payload := mchan.Send(h.AesKey, path, data)
	resp, err := http.Post(h.Host, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return mchan.DecodeReply(h.AesKey, body, ret)
}

func (c *HTTP) initClock() error {
	var timestamp int64
	if err := c.httpReq(&timestamp, "/time", nil); err != nil {
		return err
	}
	c.clock = clock.NewByRemote(timestamp)
	logex.Info("remote time:", c.clock.Now())
	return nil
}

func (c *HTTP) Login(onLogin func(*uc.AuthResponse) error) (*uc.AuthResponse, error) {
	if err := c.initClock(); err != nil {
		return nil, logex.Trace(err)
	}

	if err := c.initClock(); err != nil {
		return nil, logex.Trace(err)
	}

	ret, err := c.doLogin(c.User, c.Pswd)
	if err != nil {
		return nil, logex.Trace(err)
	}

	if onLogin != nil {
		if err := onLogin(ret); err != nil {
			return nil, logex.Trace(err)
		}
	}

	return ret, nil
}

func (c *HTTP) doLogin(username string, password string) (*uc.AuthResponse, error) {
	req := uc.NewAuthRequest(
		username, c.clock.Unix(), []byte(password), c.AesKey)
	var ret uc.AuthResponse
	if err := c.httpReq(&ret, "/auth", req); err != nil {
		return nil, err
	}
	return &ret, nil
}

type replyError struct {
	Error string `json:"error"`
}
