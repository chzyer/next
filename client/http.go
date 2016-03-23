package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"gopkg.in/logex.v1"

	"github.com/chzyer/next/uc"
	"github.com/chzyer/next/util/clock"
)

func (c *Client) httpReq(ret interface{}, path string, data interface{}) error {
	var resp *http.Response
	var err error
	if data == nil {
		resp, err = http.Get(c.cfg.Host + path)
	} else {
		jsonBody, err := json.Marshal(data)
		if err != nil {
			return err
		}
		body := bytes.NewReader(jsonBody)
		resp, err = http.Post(c.cfg.Host+path, "application/json", body)
	}
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		var err replyError
		json.Unmarshal(body, &err)
		return fmt.Errorf(err.Error)
	}
	if err := json.Unmarshal(body, ret); err != nil {
		return err
	}
	return nil
}

func (c *Client) initClock() error {
	var timestamp int64
	if err := c.httpReq(&timestamp, "/time", nil); err != nil {
		return err
	}
	c.clock = clock.NewByRemote(timestamp)
	logex.Info("remote time:", c.clock.Now())
	return nil
}

func (c *Client) Login(username string, password string) (*uc.AuthResponse, error) {
	req := uc.NewAuthRequest(
		username, c.clock.Unix(), []byte(password), []byte(c.cfg.AesKey))
	var ret uc.AuthResponse
	if err := c.httpReq(&ret, "/auth", req); err != nil {
		return nil, err
	}
	return &ret, nil
}

type replyError struct {
	Error string `json:"error"`
}
