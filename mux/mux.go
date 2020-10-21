package mux

import (
	"net"
	"sync"
)

const (
	HandshakeBufferSize = 1024
	ReadBufferSize      = 1024
	WriteBufferSize     = 1024
)

type Mux struct {
	privateKey PrivateKey

	listenLock sync.Mutex
}

func New(pk PrivateKey) *Mux {
	return &Mux{
		privateKey: pk,
	}
}

func (m *Mux) Listen(l net.Listener) error {
	m.listenLock.Lock()
	defer m.listenLock.Unlock()
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
	}
}
