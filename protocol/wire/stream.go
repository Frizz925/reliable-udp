package wire

import (
	"errors"
	"io"
	"reliable-udp/protocol/frame"
	"reliable-udp/protocol/wire/interop"
	"sync"
)

var ErrStreamAlreadyClosed = errors.New("stream already closed")

type Stream struct {
	peer    *Peer
	interop *interop.Stream
	mu      sync.Mutex
	closed  bool
}

var _ io.ReadWriteCloser = (*Stream)(nil)

func NewStream(peer *Peer, interop *interop.Stream) *Stream {
	return &Stream{
		peer:    peer,
		interop: interop,
	}
}

func (s *Stream) StreamID() frame.StreamID {
	return s.interop.StreamID()
}

func (s *Stream) Read(b []byte) (int, error) {
	return 0, nil
}

func (s *Stream) Write(b []byte) (int, error) {
	return 0, nil
}

func (s *Stream) Close() error {
	return s.close(true)
}

func (s *Stream) close(remove bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrStreamAlreadyClosed
	}
	if !s.interop.Closed() {
		if err := s.interop.Close(); err != nil {
			return err
		}
	}
	if remove {
		s.peer.remove(s.StreamID())
	}
	return nil
}
