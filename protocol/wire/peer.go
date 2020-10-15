package wire

import (
	"errors"
	"reliable-udp/protocol/frame"
	"reliable-udp/protocol/wire/interop"
	"sync"
)

var (
	ErrPeerAlreadyClosed = errors.New("peer already closed")
)

type Peer struct {
	listener *Listener
	interop  *interop.Peer
	mu       sync.Mutex
	streams  map[frame.StreamID]*Stream
	nextId   frame.StreamID
	closed   bool
}

func NewPeer(l *Listener, p *interop.Peer) *Peer {
	return &Peer{
		listener: l,
		interop:  p,
		streams:  make(map[frame.StreamID]*Stream),
	}
}

func (p *Peer) Close() error {
	return p.close(true)
}

func (p *Peer) Stream() *Stream {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nextId++
	sid := p.nextId
	s, ok := p.streams[sid]
	if !ok {
		s = NewStream(p, p.interop.Stream(sid))
		p.streams[sid] = s
	}
	return s
}

func (p *Peer) close(remove bool) error {
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
	if !p.interop.Closed() {
		if err := p.interop.Close(); err != nil {
			return err
		}
	}
	if remove {
		addr := p.interop.RemoteAddr().String()
		p.listener.remove(addr)
	}
	p.listener = nil
	p.streams = nil
	p.interop = nil
	p.closed = true
	return nil
}

func (p *Peer) remove(sid frame.StreamID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.streams, sid)
}
