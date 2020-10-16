package mux

import (
	"errors"
	"net"
	"reliable-udp/mux/frame"
	"sync"

	log "github.com/sirupsen/logrus"
)

const peerAcceptBacklog = 512

var ErrMuxClosed = errors.New("mux closed")

type Mux struct {
	conn  UDPConn
	peers map[string]*Peer

	mu  sync.RWMutex
	amu sync.Mutex
	wmu sync.Mutex

	acceptCh chan Packet
	errorCh  chan error

	closed bool
}

func New(conn UDPConn) *Mux {
	m := &Mux{
		conn:     conn,
		peers:    make(map[string]*Peer),
		acceptCh: make(chan Packet, peerAcceptBacklog),
		errorCh:  make(chan error, 1),
	}
	go m.listen()
	return m
}

func (m *Mux) Accept() (*Peer, error) {
	if err := m.errIfClosed(); err != nil {
		return nil, err
	}
	m.amu.Lock()
	defer m.amu.Unlock()
	select {
	case pa := <-m.acceptCh:
		return m.Peer(pa.RemoteAddr), nil
	case err := <-m.errorCh:
		return nil, err
	}
}

func (m *Mux) Peer(raddr *net.UDPAddr) *Peer {
	if m.Closed() {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	addr := raddr.String()
	p, ok := m.peers[addr]
	if !ok {
		p = NewPeer(m, raddr)
		m.peers[addr] = p
	}
	return p
}

func (m *Mux) Closed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

func (m *Mux) remove(addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.peers, addr)
}

func (m *Mux) write(raddr *net.UDPAddr, b []byte) (int, error) {
	m.wmu.Lock()
	defer m.wmu.Unlock()
	return m.conn.WriteToUDP(b, raddr)
}

func (m *Mux) listen() {
	if err := m.listenLoop(); err != nil {
		log.Error(err)
	}
}

func (m *Mux) listenLoop() error {
	defer m.close()
	b := make([]byte, packetMaxSize)
	for {
		n, raddr, err := m.conn.ReadFromUDP(b)
		if err != nil {
			m.errorCh <- err
			return err
		}
		fr, err := frame.Decode(b[:n])
		if err != nil {
			// Silently ignore errors
			continue
		}
		pa := NewPacket(*fr, raddr)
		if pa.IsSYN() {
			m.handleSYN(pa)
		}
		m.dispatch(pa)
	}
}

func (m *Mux) handleSYN(pa Packet) {
	m.mu.RLock()
	addr := pa.RemoteAddr.String()
	_, ok := m.peers[addr]
	m.mu.RUnlock()
	if !ok {
		m.acceptCh <- pa
	}
}

func (m *Mux) dispatch(pa Packet) {
	m.mu.RLock()
	addr := pa.RemoteAddr.String()
	p, ok := m.peers[addr]
	m.mu.RUnlock()
	if !ok {
		return
	}
	p.dispatch(pa)
}

func (m *Mux) close() {
	if m.Closed() {
		return
	}
	for _, p := range m.peers {
		p.close(false)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conn = nil
	m.peers = nil
	m.closed = false
}

func (m *Mux) errIfClosed() error {
	if m.Closed() {
		return ErrMuxClosed
	}
	return nil
}
