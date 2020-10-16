package mux

import (
	"errors"
	"net"
	"sync"
)

const streamAcceptBacklog = 256

var (
	ErrPeerInterrupted = errors.New("peer interrupted")
	ErrPeerClosed      = errors.New("peer closed")
)

type Peer struct {
	mux     *Mux
	raddr   *net.UDPAddr
	streams map[uint32]*Stream

	mu  sync.RWMutex
	amu sync.Mutex

	acceptCh chan Packet
	die      chan struct{}

	closed bool
}

func NewPeer(mux *Mux, raddr *net.UDPAddr) *Peer {
	return &Peer{
		mux:      mux,
		raddr:    raddr,
		acceptCh: make(chan Packet, streamAcceptBacklog),
		die:      make(chan struct{}),
	}
}

func (p *Peer) Accept() (*Stream, error) {
	if err := p.errIfClosed(); err != nil {
		return nil, err
	}
	p.amu.Lock()
	defer p.amu.Unlock()
	select {
	case pa := <-p.acceptCh:
		return p.Stream(pa.StreamID), nil
	case <-p.die:
		return nil, ErrPeerInterrupted
	}
}

func (p *Peer) Stream(sid uint32) *Stream {
	if p.Closed() {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.streams[sid]
	if !ok {
		s = NewStream(p, sid)
		p.streams[sid] = s
	}
	return s
}

func (p *Peer) Closed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}

func (p *Peer) Close() error {
	return p.close(true)
}

func (p *Peer) remove(sid uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.streams, sid)
}

func (p *Peer) write(b []byte) (int, error) {
	return p.mux.write(p.raddr, b)
}

func (p *Peer) dispatch(pa Packet) {
	if p.Closed() {
		return
	}
	if pa.IsSYN() {
		p.handleSYN(pa)
	}
	p.mu.RLock()
	s, ok := p.streams[pa.StreamID]
	p.mu.RUnlock()
	if ok {
		s.dispatch(pa)
	}
}

func (p *Peer) handleSYN(pa Packet) {
	p.mu.RLock()
	_, ok := p.streams[pa.StreamID]
	p.mu.RUnlock()
	if ok {
		p.acceptCh <- pa
	}
}

func (p *Peer) close(remove bool) error {
	if err := p.errIfClosed(); err != nil {
		return err
	}
	if remove {
		p.mux.remove(p.raddr.String())
	}
	close(p.die)
	for _, st := range p.streams {
		if err := st.close(false); err != nil {
			return err
		}
	}
	p.mu.Lock()
	p.mux = nil
	p.streams = nil
	p.closed = true
	p.mu.Unlock()
	return nil
}

func (p *Peer) errIfClosed() error {
	if p.Closed() {
		return ErrPeerClosed
	}
	return nil
}
