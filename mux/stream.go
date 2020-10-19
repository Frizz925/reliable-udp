package mux

import (
	"errors"
	"reliable-udp/mux/frame"
	"sync"
)

const streamReadBacklog = 128

var ErrStreamClosed = errors.New("stream closed")

type Stream struct {
	session *Session

	readCh  chan frame.Frame
	errorCh chan error
	die     chan struct{}

	mu   sync.Mutex
	rwmu sync.Mutex

	closed bool
}

func NewStream(session *Session) *Stream {
	return &Stream{
		session: session,
		readCh:  make(chan frame.Frame, streamReadBacklog),
		errorCh: make(chan error, 1),
		die:     make(chan struct{}),
	}
}

func (s *Stream) close(detach bool, notify bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrStreamClosed
	}

	close(s.die)
	s.closed = true
	return nil
}
