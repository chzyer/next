package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"gopkg.in/logex.v1"

	"github.com/chzyer/next/util/clock"
)

func (c *Client) httpReq(ret interface{}, path string) error {
	resp, err := http.Get(c.cfg.RemoteHost + path)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if err := json.Unmarshal(body, ret); err != nil {
		return err
	}
	return nil
}

func (c *Client) initClock() error {
	var timestamp int64
	if err := c.httpReq(&timestamp, "/time"); err != nil {
		return err
	}
	c.clock = clock.NewByRemote(timestamp)
	logex.Info("remote time:", c.clock.Now())
	return nil
}

func (c *Client) Login(username string, password string) {
	// uc.NewAuthRequest()
}
