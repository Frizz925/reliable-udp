package mux

import (
	"net"
	"reliable-udp/protocol"
)

type Session struct {
	conn *net.UDPConn
	cid  protocol.ConnectionID
}

func NewSession(conn *net.UDPConn) *Session {
	return &Session{
		conn: conn,
	}
}

func (s *Session) Serve() error {
	return nil
}
