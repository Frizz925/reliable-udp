package mux

import (
	"crypto/rsa"
	"errors"
	"net"
	"reliable-udp/mux/frame"
)

var ErrSessionClosed = errors.New("session closed")

type Session struct {
	mux   *Mux
	raddr *net.UDPAddr

	pubKey *rsa.PublicKey

	errorCh  chan error
	acceptCh chan *Stream

	closed bool
}

func NewSession(mux *Mux, raddr *net.UDPAddr) *Session {
	return &Session{
		mux:     mux,
		raddr:   raddr,
		errorCh: make(chan error, 1),
	}
}

func (s *Session) dispatch(f frame.Frame) {
	if f.IsFIN() && f.StreamID == 0 {
		if err := s.close(true, true); err != nil {
			s.errorCh <- err
		}
	}
}

func (s *Session) close(detach bool, notify bool) error {

}
