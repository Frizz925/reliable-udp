package wire

import (
	"errors"
	"io"
	"sync"
)

var ErrStreamAlreadyClosed = errors.New("stream already closed")

type Stream struct {
	peer   *Peer
	id     uint32
	mu     sync.Mutex
	closed bool
}

var _ io.ReadWriteCloser = (*Stream)(nil)

func NewStream(peer *Peer, id uint32) *Stream {
	return &Stream{
		peer: peer,
		id:   id,
	}
}

func (s *Stream) Read(b []byte) (int, error) {
	return 0, nil
}

func (s *Stream) Write(b []byte) (int, error) {
	return 0, nil
}

func (s *Stream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrStreamAlreadyClosed
	}
	s.peer.remove(s.id)
	return nil
}
