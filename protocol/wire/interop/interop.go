package interop

import (
	"errors"
	"net"
	"reliable-udp/protocol/frame"
	"reliable-udp/protocol/wire/interop/handler"
	"reliable-udp/util/observable"
	"sync"
)

var (
	ErrInteropAlreadyStarted = errors.New("interop already started")
	ErrInteropNotStarted     = errors.New("interop not started")
	ErrAcceptInterrupted     = errors.New("accept interrupted")
)

// Interop is a thin wrapper around a UDP connection.
// It is responsible for sending and receiving packets
// of data based on the defined protocols.
type Interop struct {
	UDPConn
	mu    sync.RWMutex
	ob    *observable.Observable
	peers map[string]*Peer
}

func New(conn UDPConn) *Interop {
	iop := &Interop{
		UDPConn: conn,
		peers:   make(map[string]*Peer),
		ob:      observable.New(),
	}
	iop.start()
	return iop
}

func (i *Interop) Peer(raddr *net.UDPAddr) *Peer {
	i.mu.Lock()
	defer i.mu.Unlock()
	addr := raddr.String()
	p, ok := i.peers[addr]
	if !ok {
		p = NewPeer(i, raddr)
		i.peers[addr] = p
	}
	return p
}

func (i *Interop) AcceptPeer() (*Peer, error) {
	ob := i.ob.Observe()
	defer ob.Dispose()
	h := handler.AcceptPeer(i.exists)
	ob.Handle(h)
	select {
	case raddr := <-h.RemoteAddr():
		return i.Peer(raddr), nil
	case err := <-h.Error():
		return nil, err
	}
}

func (i *Interop) exists(addr string) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	_, ok := i.peers[addr]
	return ok
}

func (i *Interop) remove(addr string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.peers, addr)
}

func (i *Interop) start() {
	go i.workerLoop()
}

func (i *Interop) workerLoop() {
	buf := frame.Buffer()
	for {
		n, raddr, err := i.ReadFromUDP(buf)
		evt := handler.NewEvent(raddr, err)
		if err != nil {
			i.ob.Dispatch(evt)
			return
		}
		evt.Frame, evt.Error = frame.Decode(buf[:n])
		if evt.Error == nil {
			i.ob.Dispatch(evt)
			i.mu.RLock()
			p := i.peers[raddr.String()]
			i.mu.RUnlock()
			if p != nil {
				p.ob.Dispatch(evt)
			}
		}
	}
}
