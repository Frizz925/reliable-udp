package mux

import (
	"crypto/rsa"
	"net"
	"reliable-udp/mux/frame"
	"sync"
)

type Mux struct {
	conn     *net.UDPConn
	privKey  *rsa.PrivateKey
	sessions map[string]*Session

	mu sync.RWMutex

	lastError error
}

func New(conn *net.UDPConn, privKey *rsa.PrivateKey) *Mux {
	m := &Mux{
		conn:     conn,
		privKey:  privKey,
		sessions: make(map[string]*Session),
	}
	go m.recv()
	return m
}

func (m *Mux) Error() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastError
}

func (m *Mux) send(b []byte, raddr *net.UDPAddr) (int, error) {
	return m.conn.WriteToUDP(b, raddr)
}

func (m *Mux) recv() {
	b := make([]byte, 65535)
	for {
		n, raddr, err := m.conn.ReadFromUDP(b)
		if err != nil {
			m.mu.Lock()
			m.lastError = err
			m.mu.Unlock()
			return
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
}
