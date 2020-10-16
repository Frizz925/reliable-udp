package mux

import (
	"errors"
	"io"
	"net"
	"reliable-udp/mux/frame"
	"sync"
)

const (
	packetBufferSize     = 65535
	sessionAcceptBacklog = 512
)

var ErrMuxClosed = errors.New("mux closed")

type Mux struct {
	conn     *net.UDPConn
	sessions map[string]*Session

	acceptCh chan *Session
	errorCh  chan error
	die      chan struct{}

	mu  sync.RWMutex
	amu sync.Mutex

	closed bool
}

func New(laddr *net.UDPAddr) (*Mux, error) {
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, err
	}
	m := &Mux{
		conn:     conn,
		sessions: make(map[string]*Session),
		acceptCh: make(chan *Session, sessionAcceptBacklog),
		errorCh:  make(chan error, 1),
		die:      make(chan struct{}),
	}
	go m.recv()
	return m, nil
}

func (m *Mux) LocalAddr() *net.UDPAddr {
	return m.conn.LocalAddr().(*net.UDPAddr)
}

func (m *Mux) Accept() (*Session, error) {
	if err := m.errIfClosed(); err != nil {
		return nil, err
	}
	m.amu.Lock()
	defer m.amu.Unlock()
	select {
	case s := <-m.acceptCh:
		return s, nil
	case err := <-m.errorCh:
		return nil, err
	case <-m.die:
		return nil, io.EOF
	}
}

func (m *Mux) Session(raddr *net.UDPAddr) (*Session, error) {
	if err := m.errIfClosed(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	addr := raddr.String()
	s, ok := m.sessions[addr]
	if !ok {
		s = NewSession(m, raddr)
		m.sessions[addr] = s
	}
	return s, nil
}

func (m *Mux) Write(b []byte, raddr *net.UDPAddr) (int, error) {
	return m.conn.WriteToUDP(b, raddr)
}

func (m *Mux) Closed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

func (m *Mux) Close() error {
	if err := m.errIfClosed(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if err := s.close(false, true); err != nil {
			return err
		}
	}
	if err := m.conn.Close(); err != nil {
		return err
	}
	m.sessions = nil
	m.closed = true
	return nil
}

func (m *Mux) remove(addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		delete(m.sessions, addr)
	}
}

func (m *Mux) recv() {
	if err := m.recvLoop(); err != nil {
		m.errorCh <- err
	}
}

func (m *Mux) recvLoop() error {
	b := make([]byte, packetBufferSize)
	for {
		n, raddr, err := m.conn.ReadFromUDP(b)
		if err != nil {
			return err
		}
		f, err := frame.Decode(b[:n])
		if err != nil {
			// Silently ignore decode errors
			continue
		}
		m.dispatch(*f, raddr)
	}
}

func (m *Mux) dispatch(f frame.Frame, raddr *net.UDPAddr) {
	if f.IsSYN() {
		m.handleSYN(f, raddr)
	} else {
		m.handleStream(f, raddr)
	}
}

func (m *Mux) handleSYN(f frame.Frame, raddr *net.UDPAddr) {
	m.mu.RLock()
	addr := raddr.String()
	s, ok := m.sessions[addr]
	m.mu.RUnlock()
	if !ok {
		s = NewSession(m, raddr)
		m.mu.Lock()
		m.sessions[addr] = s
		m.mu.Unlock()
		m.acceptCh <- s
	}
	s.dispatch(f)
}

func (m *Mux) handleStream(f frame.Frame, raddr *net.UDPAddr) {
	m.mu.RLock()
	s := m.sessions[raddr.String()]
	m.mu.RUnlock()
	if s != nil {
		s.dispatch(f)
	}
}

func (m *Mux) errIfClosed() error {
	if m.Closed() {
		return ErrMuxClosed
	}
	return nil
}
