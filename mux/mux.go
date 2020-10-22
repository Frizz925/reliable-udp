package mux

import (
	"bytes"
	"crypto/cipher"
	"net"
	"reliable-udp/protocol"
	"sync"

	log "github.com/sirupsen/logrus"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	HandshakeBufferSize = 1024
	ReadBufferSize      = 1024
	WriteBufferSize     = 1024
)

type Mux struct {
	privateKey protocol.PrivateKey

	listenLock sync.Mutex
}

func New(pk protocol.PrivateKey) *Mux {
	return &Mux{
		privateKey: pk,
	}
}

func (m *Mux) Listen(conn *net.UDPConn) error {
	m.listenLock.Lock()
	defer m.listenLock.Unlock()
	aead, err := chacha20poly1305.New(m.privateKey[:])
	if err != nil {
		return err
	}
	b := protocol.NewBuffer()
	buf := bytes.NewBuffer(protocol.EmptyBuffer())
	for {
		n, _, err := conn.ReadFromUDP(b)
		if err != nil {
			return err
		}
		buf.Reset()
		buf.Write(b[:n])
		p, err := protocol.Deserialize(buf)
		if err != nil {
			// Silently ignore packets that fail to be deserialized
			continue
		}
		if err := logPacket(aead, p); err != nil {
			log.Errorf("Error logging packet: %+v", err)
		}
	}
}

func logPacket(aead cipher.AEAD, p protocol.Packet) error {
	f := p.Frame
	if f.Type() == protocol.FrameCrypto {
		nonce := p.Nonce()
		raw := f.(*protocol.Raw)
		cf, err := protocol.DecryptFrame(aead, nonce, raw)
		if err != nil {
			return err
		}
		f = cf
	}
	log.Debug(f)
	if cf, ok := f.(*protocol.Crypto); ok {
		f = cf.Frame()
	}
	if raw, ok := f.(*protocol.Raw); ok {
		content := raw.Content()
		log.Debugf("Raw content: %s", string(content))
	}
	return nil
}
