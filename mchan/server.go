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

func (s *Server) write(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Length", fmt.Sprintf("%v", len(data)))
	n, err := w.Write(data)
	logex.Debugf("write %v bytes", n)
	if n != len(data) {
		logex.Error("short write: %v, %v", n, len(data))
		return
	}
	if err != nil {
		logex.Error("on write error:", err)
	}
	return
}

func (s *Server) DecodeRequest(w http.ResponseWriter, body []byte) error {
	reply, err := Decode(s.key, body)
	if err != nil {
		return err
	}
	if f := s.router[reply.Path]; f != nil {
		ret := f(&Req{reply.Payload})
		logex.Debug("got reply:", ret)
		if ret == nil {
			return nil
		}
		switch t := ret.(type) {
		case error:
			s.write(w, ReplyError(s.key, t))
		default:
			s.write(w, Reply(s.key, ret))
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
