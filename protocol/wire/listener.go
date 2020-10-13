package wire

import (
	"errors"
	"net"
	"sync"
)

var (
	ErrListenerNotOpen     = errors.New("listener not open")
	ErrListenerAlreadyOpen = errors.New("listener already open")
)

type Listener struct {
	conn  *net.UDPConn
	mu    sync.Mutex
	peers map[string]*Peer
	open  bool
}

func NewListener() *Listener {
	return &Listener{}
}

func (l *Listener) Open(laddr *net.UDPAddr) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.open {
		return ErrListenerAlreadyOpen
	}
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}
	l.conn = conn
	l.peers = make(map[string]*Peer)
	l.open = true
	return nil
}

func (l *Listener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.open {
		return ErrListenerNotOpen
	}
	for _, peer := range l.peers {
		if err := peer.close(); err != nil {
			return err
		}
	}
	if err := l.conn.Close(); err != nil {
		return err
	}
	l.conn = nil
	l.peers = nil
	l.open = false
	return nil
}

func (l *Listener) Peer(addr string) (*Peer, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.open {
		return nil, ErrListenerNotOpen
	}
	p, ok := l.peers[addr]
	if !ok {
		laddr := l.LocalAddr()
		raddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, err
		}
		p = NewPeer(l, addr)
		if err := p.open(laddr, raddr); err != nil {
			return nil, err
		}
		l.peers[addr] = p
	}
	return p, nil
}

func (l *Listener) LocalAddr() *net.UDPAddr {
	if l.conn != nil {
		addr := l.conn.LocalAddr()
		if laddr, ok := addr.(*net.UDPAddr); ok {
			return laddr
		}
	}
	return nil
}

func (l *Listener) remove(addr string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.peers, addr)
}
