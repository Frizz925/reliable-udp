package wire

import (
	"errors"
	"io"
	"net"
	"reliable-udp/protocol/wire/interop"
	"sync"
)

var (
	ErrListenerNotOpen     = errors.New("listener not open")
	ErrListenerAlreadyOpen = errors.New("listener already open")
)

type Listener struct {
	conn    *net.UDPConn
	interop *interop.Interop
	mu      sync.Mutex
	peers   map[string]*Peer
	open    bool
}

func NewListener() *Listener {
	return &Listener{}
}

func Listen(addr string) (*Listener, error) {
	l := NewListener()
	laddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	if err := l.Open(laddr); err != nil {
		return nil, err
	}
	return l, nil
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
	l.interop = interop.New(conn)
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
		if err := peer.close(false); err != nil {
			return err
		}
	}
	if err := l.conn.Close(); err != nil {
		return err
	}
	l.conn = nil
	l.interop = nil
	l.peers = nil
	l.open = false
	return nil
}

func (l *Listener) Accept() (*Peer, error) {
	a := l.interop.Accept()
	if a == nil {
		return nil, io.EOF
	}
	addr := a.RemoteAddr().String()
	return l.Peer(addr)
}

func (l *Listener) Peer(addr string) (*Peer, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.open {
		return nil, ErrListenerNotOpen
	}
	p, ok := l.peers[addr]
	if !ok {
		raddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, err
		}
		p = NewPeer(l, l.interop.Peer(raddr))
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
