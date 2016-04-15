package controller

import (
	"encoding/json"

	"github.com/chzyer/flow"
	"github.com/chzyer/next/packet"
	"github.com/chzyer/next/uc"
)

type Server struct {
	*Controller
	flow  *flow.Flow
	user  *uc.User
	toTun chan<- []byte
	ports []int
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

func (s *Server) NotifyDataChannel(port []int) {
	s.ports = port
	return
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
			case packet.NEWDC:
				ret, _ := json.Marshal(s.ports)
				s.Send(p.Reply(ret))
				continue
			case packet.DATA:
				select {
				case s.toTun <- p.Data():
				case <-s.flow.IsClose():
					break loop
				}
			}
			if p.Type.IsReq() {
				s.Send(p.Reply(nil))
			}
		case <-s.flow.IsClose():
			break loop
		}
	}
}

func (s *Server) UserRelogin(u *uc.User) {

}
