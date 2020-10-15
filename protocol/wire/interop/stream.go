package interop

import (
	"errors"
	"reliable-udp/protocol/frame"
	"sync"
)

var ErrStreamAlreadyClosed = errors.New("stream already closed")

type Stream struct {
	peer   *Peer
	sid    frame.StreamID
	mu     sync.RWMutex
	closed bool
}

func NewStream(peer *Peer, sid frame.StreamID) *Stream {
	return &Stream{
		peer: peer,
		sid:  sid,
	}
}

func (s *Stream) StreamID() frame.StreamID {
	return s.sid
}

func (s *Stream) Send(data frame.Data) error {
	return s.peer.Send(data)
}

func (s *Stream) Handshake(length uint16, hash []byte) error {
	return s.Send(frame.Handshake{
		StreamID: s.sid,
		Length:   length,
		Hash:     hash,
	})
}

func (s *Stream) AckHandshake(size uint16) error {
	return s.Send(frame.HandshakeAck{
		StreamID: s.sid,
		Size:     size,
	})
}

func (s *Stream) Stream(seq uint16, off uint16, chunk []byte) error {
	return s.Send(frame.Stream{
		StreamID: s.sid,
		Sequence: seq,
		Offset:   off,
		Chunk:    chunk,
	})
}

func (s *Stream) AckStream(seq uint16) error {
	return s.Send(frame.StreamAck{
		StreamID: s.sid,
		Sequence: seq,
	})
}

func (s *Stream) Close() error {
	if err := s.Send(frame.Fin{StreamID: s.sid}); err != nil {
		return err
	}
	return s.close(true)
}

func (s *Stream) Closed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

func (s *Stream) close(remove bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrStreamAlreadyClosed
	}
	if remove {
		s.peer.remove(s.sid)
	}
	s.peer = nil
	s.closed = true
	return nil
}
