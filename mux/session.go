package mux

import (
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"reliable-udp/mux/frame"
	"sync"

	log "github.com/sirupsen/logrus"
)

const streamAcceptBacklog = 256

var (
	ErrSessionClosed    = errors.New("peer closed")
	ErrStreamsExhausted = errors.New("streams exhausted")
)

type Session struct {
	mux     *Mux
	raddr   *net.UDPAddr
	streams map[uint32]*Stream
	nextId  uint32

	mu  sync.RWMutex
	amu sync.Mutex

	acceptCh chan *Stream
	errorCh  chan error
	die      chan struct{}

	closed bool
}

func NewSession(mux *Mux, raddr *net.UDPAddr) *Session {
	return &Session{
		mux:      mux,
		raddr:    raddr,
		streams:  make(map[uint32]*Stream),
		acceptCh: make(chan *Stream, streamAcceptBacklog),
		errorCh:  make(chan error, 1),
		die:      make(chan struct{}),
	}
}

func (s *Session) LocalAddr() *net.UDPAddr {
	return s.mux.LocalAddr()
}

func (s *Session) RemoteAddr() *net.UDPAddr {
	return s.raddr
}

func (s *Session) Accept() (*Stream, error) {
	if err := s.errIfClosed(); err != nil {
		return nil, err
	}
	s.amu.Lock()
	defer s.amu.Unlock()
	select {
	case s := <-s.acceptCh:
		return s, nil
	case err := <-s.errorCh:
		return nil, err
	case <-s.die:
		return nil, io.EOF
	}
}

func (s *Session) OpenStream() (*Stream, error) {
	if err := s.errIfClosed(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sid := s.nextId
	for {
		if sid >= math.MaxUint32-1 {
			return nil, ErrStreamsExhausted
		}
		sid++
		_, ok := s.streams[sid]
		if !ok {
			break
		}
	}
	s.nextId = sid
	conn := NewStream(s, sid)
	s.streams[sid] = conn
	return conn, nil
}

func (s *Session) Write(b []byte) (int, error) {
	return s.mux.Write(b, s.raddr)
}

func (s *Session) Closed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

func (s *Session) Close() error {
	return s.close(true, true)
}

func (s *Session) String() string {
	return fmt.Sprintf("%s <-> %s", s.LocalAddr(), s.RemoteAddr())
}

func (s *Session) remove(sid uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.streams, sid)
}

func (s *Session) writeFin() error {
	f := frame.New(frame.FlagFIN, 0, nil)
	_, err := s.Write(f.Bytes())
	return err
}

func (s *Session) dispatch(f frame.Frame) {
	log.Debugf("[%s] %5s: %s", s, "read", f)
	if f.IsFIN() && f.StreamID == 0 {
		if err := s.close(true, false); err != nil {
			s.errorCh <- err
		}
	} else if f.IsSYN() {
		s.handleSYN(f)
	} else {
		s.handleStream(f)
	}
}

func (s *Session) handleSYN(f frame.Frame) {
	s.mu.RLock()
	conn, ok := s.streams[f.StreamID]
	s.mu.RUnlock()
	if !ok {
		conn = NewStream(s, f.StreamID)
		s.mu.Lock()
		s.streams[f.StreamID] = conn
		s.mu.Unlock()
		s.acceptCh <- conn
	}
	conn.dispatch(f)
}

func (s *Session) handleStream(f frame.Frame) {
	s.mu.RLock()
	conn, ok := s.streams[f.StreamID]
	s.mu.RUnlock()
	if ok {
		conn.dispatch(f)
	}
}

func (s *Session) close(detach bool, notify bool) error {
	if err := s.errIfClosed(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, conn := range s.streams {
		if err := conn.close(false, notify); err != nil {
			return err
		}
	}
	if err := s.writeFin(); err != nil {
		return err
	}
	close(s.die)
	if detach {
		s.mux.remove(s.RemoteAddr().String())
	}
	s.streams = nil
	s.closed = true
	return nil
}

func (s *Session) errIfClosed() error {
	if s.Closed() {
		return ErrSessionClosed
	}
	return nil
}
