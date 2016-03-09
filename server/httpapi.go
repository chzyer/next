package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/chzyer/next/uc"
	"github.com/chzyer/next/util/clock"
)

type HttpApiConfig struct {
	AesKey []byte

	CertFile string
	KeyFile  string
}

type HttpApi struct {
	listen string
	cfg    *HttpApiConfig
	users  *uc.Users
	clock  *clock.Clock
	server *http.Server
}

func NewHttpApi(listen string, users *uc.Users, ct *clock.Clock, cfg *HttpApiConfig) *HttpApi {
	server := &http.Server{
		Addr:           listen,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 10,
	}
	return &HttpApi{
		cfg:    cfg,
		clock:  ct,
		server: server,
		listen: listen,
		users:  users,
	}
}

type replyError struct {
	Error string `json:"error"`
}

func (h *HttpApi) replyError(w http.ResponseWriter, err interface{}) {
	w.WriteHeader(400)
	switch t := err.(type) {
	case error:
		h.reply(w, replyError{t.Error()})
	case string:
		h.reply(w, replyError{t})
	}
}

func (h *HttpApi) reply(w http.ResponseWriter, obj interface{}) {
	ret, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	w.Write(ret)
}

func (h *HttpApi) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", h.Auth)
	mux.HandleFunc("/time", h.Time)
	h.server.Handler = mux
	return h.server.ListenAndServe()
}
