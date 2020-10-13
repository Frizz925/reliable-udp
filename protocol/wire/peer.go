package wire

import (
	"errors"
	"net"
	"sync"
)

var (
	ErrPeerAlreadyClosed = errors.New("peer already closed")
)

type Peer struct {
	listener *Listener
	addr     string
	interop  *Interop
	mu       sync.Mutex
	streams  map[uint32]*Stream
	nextId   uint32
	closed   bool
}

func NewPeer(l *Listener, addr string) *Peer {
	return &Peer{
		listener: l,
		addr:     addr,
		streams:  make(map[uint32]*Stream),
		nextId:   0,
		closed:   false,
	}
}

func (p *Peer) Close() error {
	if err := p.close(); err != nil {
		return err
	}
	p.listener.remove(p.addr)
	return nil
}

func (p *Peer) Stream() *Stream {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nextId++
	id := p.nextId
	s, ok := p.streams[id]
	if !ok {
		s = NewStream(p, id)
		p.streams[id] = s
	}
	return s
}

func (p *Peer) open(laddr *net.UDPAddr, raddr *net.UDPAddr) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	conn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		return err
	}
	p.interop = NewInterop(conn)
	return nil
}

func (p *Peer) close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrPeerAlreadyClosed
	}
	for _, st := range p.streams {
		if err := st.Close(); err != nil {
			return err
		}
	}
	if err := p.interop.Close(); err != nil {
		return err
	}
	p.listener = nil
	p.streams = nil
	p.interop = nil
	p.closed = true
	return nil
}

func (p *Peer) remove(id uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.streams, id)
}
