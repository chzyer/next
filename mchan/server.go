package mchan

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"gopkg.in/logex.v1"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/util/clock"
)

type Server struct {
	cfg    *SvrConf
	flow   *flow.Flow
	key    []byte
	server *http.Server
	ct     *clock.Clock
	router map[string]HandlerFunc
}

type SvrConf struct {
	CertFile string
	KeyFile  string
}

func NewServer(f *flow.Flow, listen string, ct *clock.Clock, key []byte, cfg *SvrConf) *Server {
	httpserver := &http.Server{
		Addr:           listen,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 10,
	}
	httpserver.SetKeepAlivesEnabled(false)
	svr := &Server{
		flow:   f,
		cfg:    cfg,
		key:    key,
		server: httpserver,
		ct:     ct,
		router: make(map[string]HandlerFunc),
	}
	svr.server.Handler = svr
	return svr
}

type Req struct {
	payload []byte
}

func (r *Req) Unmarshal(obj interface{}) error {
	return json.Unmarshal(r.payload, obj)
}

type HandlerFunc func(*Req) interface{}

func (s *Server) HandleFunc(path string, f HandlerFunc) {
	s.router[path] = f
}

func (s *Server) DecodeRequest(w http.ResponseWriter, body []byte) error {
	reply, err := Decode(s.key, body)
	if err != nil {
		return err
	}
	if f := s.router[reply.Path]; f != nil {
		ret := f(&Req{reply.Payload})
		if ret == nil {
			return nil
		}
		switch t := ret.(type) {
		case error:
			w.Write(ReplyError(s.key, t))
		default:
			w.Write(Reply(s.key, ret))
		}
		return nil
	}
	return fmt.Errorf("not found")
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logex.Error(err)
		return
	}
	if err := s.DecodeRequest(w, body); err == nil {
		return
	}
	logex.Warn("invalid request from:", req.RemoteAddr)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<html>
		<head><title>Hello World!</title></head>
		<body><h1>Hello</h1></body>
	</html>
`))
}

func (s *Server) Run() error {
	return s.server.ListenAndServe()
}
