package controller

import (
	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/uc"
)

type Server struct {
	*Controller
	flow  *flow.Flow
	user  *uc.User
	toTun chan<- []byte
}

func NewServer(f *flow.Flow, u *uc.User, toTun chan<- []byte) *Server {
	fromDC, toDC := u.GetFromController()
	ctl := NewController(f, toDC, fromDC)
	s := &Server{
		flow:       ctl.flow,
		Controller: ctl,
		user:       u,
		toTun:      toTun,
	}
	go s.loop()
	return s
}

func (s *Server) loop() {
	s.flow.Add(1)
	defer s.flow.DoneAndClose()

	out := s.Controller.GetOutChan()
loop:
	for {
		select {
		case p := <-out:
			switch p.Type {
			case packet.DATA:
				select {
				case s.toTun <- p.Data():
				case <-s.flow.IsClose():
					break loop
				}
			}
			s.Send(p.Reply(nil))
		case <-s.flow.IsClose():
			break loop
		}
	}
}

func (s *Server) UserRelogin(u *uc.User) {

}
