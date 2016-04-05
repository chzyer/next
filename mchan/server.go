package mchan

import (
	"encoding/json"
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

type HandlerFunc func(payload []byte) interface{}

func (s *Server) HandleFunc(path string, f HandlerFunc) {
	s.router[path] = f
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logex.Error(err)
		return
	}

	reply, err := Decode(s.key, body)
	if err != nil {
		logex.Error(err)
		return
	}
	if f := s.router[reply.Path]; f != nil {
		ret := f(reply.Payload)
		if ret == nil {
			return
		}
		switch t := ret.(type) {
		case error:
			w.Write(ReplyError(s.key, t))
		default:
			w.Write(Reply(s.key, ret))
		}
	}
}

func (s *Server) Run() error {
	return s.server.ListenAndServe()
}
