package server

import (
	"net"
	"os"
)

type Shell struct {
	sock string
	conn *net.UnixConn
}

func NewShell(sock string) (*Shell, error) {
	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{
		Name: sock,
		Net:  "unixgram",
	})
	if err != nil {
		return nil, err
	}
	return &Shell{
		sock: sock,
		conn: conn,
	}, nil
}

func (s *Shell) Close() {
	s.conn.Close()
	os.Remove(s.sock)
}
