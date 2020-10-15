package interop

import (
	"errors"
	"math"
	"net"
	"reliable-udp/protocol/frame"
	"reliable-udp/util/observable"
	"sync"
)

var (
	ErrPeerAlreadyClosed = errors.New("peer already closed")
	ErrStreamsExhausted  = errors.New("streams exhausted")
)

type Handshake struct {
	*Stream
	frame.Handshake
}

type Peer struct {
	interop *Interop
	raddr   *net.UDPAddr
	ob      *observable.Observable

	streams map[frame.StreamID]*Stream
	nextId  frame.StreamID
	mu      sync.RWMutex

	closed bool
}

func NewPeer(interop *Interop, raddr *net.UDPAddr) *Peer {
	return &Peer{
		interop: interop,
		raddr:   raddr,
		ob:      observable.New(),
	}
}

func (p *Peer) RemoteAddr() *net.UDPAddr {
	return p.raddr
}

func (p *Peer) OpenStream() (*Stream, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	sid := p.nextId
	for {
		if sid >= math.MaxUint16-1 {
			return nil, ErrStreamsExhausted
		}
		sid++
		if _, ok := p.streams[sid]; !ok {
			break
		}
	}
	p.nextId = sid
	s := NewStream(p, sid)
	p.streams[sid] = s
	return s, nil
}

func (p *Peer) AcceptStream() (*Stream, error) {
	ob := p.ob.Observe()
	defer ob.Dispose()
	ch := make(chan interface{})
	ob.HandleFunc(func(o *ob))
	switch v := (<-ch).(type) {
	case error:
		return nil, v
	case frame.StreamID:
		return p.Stream(v), nil
	}
	return nil, ErrAcceptInterrupted
}

func (p *Peer) Stream(sid frame.StreamID) *Stream {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.streams[sid]
	if !ok {
		s = NewStream(p, sid)
		p.streams[sid] = s
	}
	return s
}

func (p *Peer) Send(data frame.Data) error {
	b := frame.New(data).Bytes()
	_, err := p.interop.WriteToUDP(b, p.raddr)
	return err
}

func (p *Peer) Close() error {
	return p.close(true)
}

func (p *Peer) Closed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}

func (p *Peer) exists(sid frame.StreamID) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.streams[sid]
	return ok
}

func (p *Peer) remove(sid frame.StreamID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.streams, sid)
}

func (p *Peer) startWorker() {
	p.once.Do(p.observeConnection)
}

func (p *Peer) observeConnection() {
	p.interop.ob.Observe().HandleFunc(func(o *observable.Observer, v interface{}) {
		e, ok := v.(event)
		if !ok {
			return
		}
		if e.raddr.String() != p.raddr.String() {
			return
		}
		p.ob.Dispatch(e)
	}, func(*observable.Observer) {
		p.ob.Dispose()
	})
}

func (p *Peer) close(remove bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrPeerAlreadyClosed
	}
	p.ob.Dispose()
	for _, s := range p.streams {
		if err := s.close(false); err != nil {
			return err
		}
	}
	if err := p.Send(frame.Fin{}); err != nil {
		return err
	}
	if remove {
		p.interop.remove(p.raddr.String())
	}
	// Remove references to avoid memory leaks
	p.interop = nil
	p.raddr = nil
	p.streams = nil
	p.ob = nil
	p.closed = true
	return nil
}
