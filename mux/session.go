package mux

import (
	"crypto/cipher"
	"errors"
	"fmt"
	"net"
	"reliable-udp/protocol"
	"reliable-udp/util"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"

	"golang.org/x/crypto/chacha20poly1305"
)

var (
	ErrSessionClosed      = errors.New("session closed")
	ErrSessionInterrupted = errors.New("session interrupted")
)

type SessionConfig struct {
	cid          protocol.ConnectionID
	key          []byte
	maxFrameSize int
	bufferSize   int
}

type Session struct {
	cfg SessionConfig

	mux  *Mux
	cid  protocol.ConnectionID
	aead cipher.AEAD
	seq  uint32

	streams struct {
		sync.RWMutex
		idMap   map[protocol.StreamID]*Stream
		openMap map[protocol.StreamID]chan<- *Stream
		id      uint32
	}

	accept chan *Stream
	die    chan struct{}
	mu     sync.Mutex

	closed util.AtomicBool
}

func NewSession(mux *Mux, cfg SessionConfig) (*Session, error) {
	aead, err := chacha20poly1305.New(cfg.key)
	if err != nil {
		return nil, err
	}
	s := &Session{
		cfg:  cfg,
		cid:  cfg.cid,
		mux:  mux,
		aead: aead,
	}

	s.streams.idMap = make(map[protocol.StreamID]*Stream)
	s.streams.openMap = make(map[protocol.StreamID]chan<- *Stream)

	s.accept = make(chan *Stream)
	s.die = make(chan struct{})

	log.Debugf("Created new session: %+v", s)
	s.closed.Set(false)
	return s, nil
}

func (s *Session) OpenStream() (*Stream, error) {
	if s.closed.Get() {
		return nil, ErrSessionClosed
	}

	var sid protocol.StreamID
	s.streams.RLock()
	for {
		sid = protocol.StreamID(s.nextId())
		_, ok := s.streams.idMap[sid]
		if !ok {
			break
		}
	}
	s.streams.RUnlock()

	ch := make(chan *Stream)
	s.streams.Lock()
	s.streams.openMap[sid] = ch
	s.streams.Unlock()

	defer func() {
		s.streams.Lock()
		delete(s.streams.openMap, sid)
		s.streams.Unlock()
		close(ch)
	}()

	openFrame := protocol.NewStreamFrame(protocol.FrameStreamOpen, sid)
	if err := s.SendStream(openFrame); err != nil {
		return nil, err
	}

	select {
	case conn := <-ch:
		return conn, nil
	case <-s.die:
		return nil, ErrSessionInterrupted
	}
}

func (s *Session) Accept() (*Stream, error) {
	if s.closed.Get() {
		return nil, ErrSessionClosed
	}
	select {
	case conn := <-s.accept:
		return conn, nil
	case <-s.die:
		return nil, ErrSessionInterrupted
	}
}

func (s *Session) LocalAddr() net.Addr {
	return s.mux.Addr()
}

func (s *Session) RemoteAddr() *net.UDPAddr {
	return s.mux.PeerAddr(s.cid)
}

func (s *Session) SendStream(frame protocol.Frame) error {
	if s.closed.Get() {
		return ErrSessionClosed
	}
	return s.mux.Send(protocol.Packet{
		ConnectionID: s.cid,
		Sequence:     s.nextSeq(),
		Type:         protocol.PacketStream,
		Frame:        frame,
	})
}

func (s *Session) Close() error {
	return s.close(true)
}

func (s *Session) String() string {
	return fmt.Sprintf("Session(CID: %d, Mux: %+v)", s.cid, s.mux)
}

func (s *Session) nextSeq() uint32 {
	return atomic.AddUint32(&s.seq, 1)
}

func (s *Session) nextId() uint32 {
	return atomic.AddUint32(&s.streams.id, 1)
}

func (s *Session) decrypt(raw protocol.Raw, nonce protocol.Nonce) (protocol.Frame, error) {
	cf, err := protocol.DecryptFrame(s.aead, nonce, raw)
	if err != nil {
		return nil, err
	}
	return cf.Frame(), nil
}

func (s *Session) encrypt(frame protocol.Frame, nonce protocol.Nonce) (protocol.Crypto, error) {
	return protocol.EncryptFrame(s.aead, nonce, frame)
}

func (s *Session) dispatch(frame protocol.Frame) {
	if s.closed.Get() {
		return
	}
	switch v := frame.(type) {
	case protocol.StreamFrame:
		switch v.Type() {
		case protocol.FrameStreamOpen:
			s.openStream(v.StreamID(), false)
		case protocol.FrameStreamAck:
			s.openStream(v.StreamID(), true)
		case protocol.FrameStreamClose:
			s.closeStream(v.StreamID())
		default:
			s.dispatchStream(v.StreamID(), v)
		}
	}
}

func (s *Session) openStream(sid protocol.StreamID, isAck bool) {
	s.streams.RLock()
	_, ok := s.streams.idMap[sid]
	s.streams.RUnlock()
	if ok {
		return
	}

	conn := NewStream(s, StreamConfig{
		SessionConfig: s.cfg,
		sid:           sid,
	})

	s.streams.Lock()
	s.streams.idMap[sid] = conn
	if isAck {
		ch, ok := s.streams.openMap[sid]
		if ok {
			delete(s.streams.openMap, sid)
			ch <- conn
		}
		s.streams.Unlock()
		return
	}
	s.streams.Unlock()

	select {
	case s.accept <- conn:
	default:
	}

	// We need to send ACK frame back for peer-initiated stream
	s.SendStream(protocol.NewStreamFrame(protocol.FrameStreamAck, sid))
}

func (s *Session) dispatchStream(sid protocol.StreamID, frame protocol.Frame) {
	s.streams.RLock()
	conn, ok := s.streams.idMap[sid]
	s.streams.RUnlock()
	if ok {
		conn.dispatch(frame)
	}
}

func (s *Session) closeStream(sid protocol.StreamID) {
	s.streams.RLock()
	conn, ok := s.streams.idMap[sid]
	s.streams.RUnlock()
	if ok {
		conn.Close()
	}
}

func (s *Session) remove(sid protocol.StreamID) {
	s.streams.Lock()
	delete(s.streams.idMap, sid)
	delete(s.streams.openMap, sid)
	s.streams.Unlock()
}

func (s *Session) close(detach bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Get() {
		return ErrSessionClosed
	}
	s.closed.Set(true)
	log.Debugf("Closing session: %+v", s)
	s.streams.RLock()
	for _, conn := range s.streams.idMap {
		if err := conn.close(false); err != nil {
			s.streams.RUnlock()
			return err
		}
	}
	s.streams.RUnlock()
	terminatePacket := protocol.Packet{
		ConnectionID: s.cid,
		Sequence:     s.nextSeq(),
		Type:         protocol.PacketTerminate,
	}
	if err := s.mux.Send(terminatePacket); err != nil {
		return err
	}
	close(s.die)
	if detach {
		s.mux.remove(s.cid)
	}
	return nil
}
