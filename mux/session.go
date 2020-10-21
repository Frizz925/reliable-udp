package mux

import (
	"net"
)

type Session struct {
	cid protocol.
}

func NewSession(conn net.Conn) *Session {
	return &Session{
		conn: conn,
	}
}

func (s *Session) Serve() error {

}