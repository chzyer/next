package dchan

import "github.com/chzyer/flow"

type Server struct {
	flow *flow.Flow
	n    int
}

func NewServer(f *flow.Flow, n int) *Server {
	s := &Server{
		flow: f,
		n:    n,
	}
	return s
}

func (s *Server) Run() {

}
