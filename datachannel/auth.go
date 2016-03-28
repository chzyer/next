package datachannel

import (
	"bytes"
	"net"
	"time"

	"github.com/chzyer/next/packet"
	"gopkg.in/logex.v1"
)

var (
	ErrInvalidUserId        = logex.Define("invalid user id")
	ErrUnexpectedPacketType = logex.Define("unexpected packet type")
)

type SvrDelegate interface {
	GetUserToken(id int) string
	GetUserChannelFromDataChannel(id int) (
		fromUser <-chan *packet.Packet, toUser chan<- *packet.Packet, err error)
}

// try resend or timeout
func ClientCheckAuth(conn net.Conn, session *packet.SessionIV) error {
	p := packet.New(session.Token, packet.Auth)
	if _, err := conn.Write(p.Marshal(session)); err != nil {
		return logex.Trace(err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	pr, err := packet.Read(session, conn)
	if err != nil {
		return logex.Trace(err)
	}
	conn.SetReadDeadline(time.Time{})

	if !bytes.Equal(pr.Payload, p.Payload) {
		return logex.NewError("invalid auth reply", pr.Payload, p.Payload)
	}
	return nil
}

func ServerCheckAuth(delegate SvrDelegate, port int, conn net.Conn) (*packet.SessionIV, error) {
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	iv, err := packet.ReadIV(conn)
	if err != nil {
		return nil, logex.Trace(err)
	}
	conn.SetReadDeadline(time.Time{})
	if int(iv.Port) != port {
		return nil, packet.ErrPortNotMatch.Trace()
	}

	token := delegate.GetUserToken(int(iv.UserId))
	if token == "" {
		return nil, ErrInvalidUserId.Trace()
	}

	s := packet.NewSessionIV(iv.UserId, iv.Port, []byte(token))
	p, err := packet.ReadWithIV(s, iv, conn)
	if err != nil {
		return nil, logex.Trace(err)
	}
	if p.Type != packet.Auth {
		return nil, ErrUnexpectedPacketType.Trace()
	}
	if !bytes.Equal(s.Token, p.Payload) {
		return nil, packet.ErrInvalidToken.Trace()
	}

	p = packet.New(s.Token, packet.AuthResp)
	if _, err := conn.Write(p.Marshal(s)); err != nil {
		return nil, logex.Trace(err)
	}
	return s, nil
}
