package mux

import (
	"errors"
	"fmt"
	"io"
	"net"
	"reliable-udp/mux/frame"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	streamReadBacklog = 128
	defaultTimeout    = 15 * time.Second
)

var (
	ErrReadTimeout = errors.New("read timeout")

	ErrStreamReset  = errors.New("stream reset")
	ErrStreamClosed = errors.New("stream closed")
)

var timerPool = &sync.Pool{
	New: func() interface{} {
		return time.NewTimer(defaultTimeout)
	},
}

type Stream struct {
	session *Session
	id      uint32

	mu   sync.RWMutex
	rwmu sync.Mutex

	readCh  chan frame.Frame
	errorCh chan error
	die     chan struct{}

	closed bool
}

func NewStream(session *Session, id uint32) *Stream {
	return &Stream{
		session: session,
		id:      id,
		readCh:  make(chan frame.Frame, streamReadBacklog),
		errorCh: make(chan error, 1),
		die:     make(chan struct{}),
	}
}

func (s *Stream) LocalAddr() *net.UDPAddr {
	return s.session.LocalAddr()
}

func (s *Stream) RemoteAddr() *net.UDPAddr {
	return s.session.RemoteAddr()
}

func (s *Stream) StreamID() uint32 {
	return s.id
}

func (s *Stream) Read(b []byte) (int, error) {
	if err := s.errIfClosed(); err != nil {
		return 0, err
	}
	s.rwmu.Lock()
	defer s.rwmu.Unlock()
	size, err := s.readInit()
	if err != nil {
		return 0, err
	}
	defer s.writeRst()
	if len(b) < size {
		return 0, io.ErrShortBuffer
	}
	if err := s.writeInitAck(); err != nil {
		return 0, err
	}
	read := 0
	for read < size {
		c, err := s.readChunk()
		if err != nil {
			return 0, err
		}
		offset := int(c.Offset)
		length := c.Length()
		for i := 0; i < length; i++ {
			b[offset+i] = c.Data[i]
		}
		if err := s.writeChunkAck(c.Sequence); err != nil {
			return 0, err
		}
		read += length
	}
	if err := s.readRst(); err != nil {
		return 0, nil
	}
	return read, nil
}

func (s *Stream) Write(b []byte) (int, error) {
	if err := s.errIfClosed(); err != nil {
		return 0, err
	}
	s.rwmu.Lock()
	defer s.rwmu.Unlock()
	if err := s.writeInit(len(b)); err != nil {
		return 0, err
	}
	defer s.writeRst()
	if err := s.readInitAck(); err != nil {
		return 0, err
	}
	seqMap := make(map[uint32]int)
	nextSeq := uint32(0)
	written := 0
	for written < len(b) {
		nextSeq++
		n, err := s.writeChunk(nextSeq, written, b[written:])
		if err != nil {
			return 0, err
		}
		seqMap[nextSeq] = written
		written += n
	}
	for len(seqMap) > 0 {
		resend := false
		seq, err := s.readChunkAck()
		if err != nil {
			if err != ErrReadTimeout {
				return 0, err
			}
			resend = true
		}
		if !resend {
			delete(seqMap, seq)
			continue
		}
		for seq, off := range seqMap {
			if _, err := s.writeChunk(seq, off, b[off:]); err != nil {
				return 0, err
			}
		}
	}
	return written, nil
}

func (s *Stream) Reset() error {
	return s.writeRst()
}

func (s *Stream) Closed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

func (s *Stream) Close() error {
	return s.close(true, true)
}

func (s *Stream) String() string {
	return fmt.Sprintf("%s <- %d -> %s", s.LocalAddr(), s.StreamID(), s.RemoteAddr())
}

func (s *Stream) readInit() (int, error) {
	f, err := s.read(frame.FlagSYN, false)
	if err != nil {
		return 0, err
	}
	init, err := frame.DecodeInit(f.Data)
	return int(init), err
}

func (s *Stream) readInitAck() error {
	flags := frame.FlagSYN | frame.FlagACK | frame.FlagRST
	f, err := s.read(flags, true)
	if err != nil {
		return err
	}
	if f.IsRST() {
		return ErrStreamReset
	}
	return nil
}

func (s *Stream) readChunk() (*frame.Chunk, error) {
	f, err := s.read(0, true)
	if err != nil {
		return nil, err
	}
	if f.IsRST() {
		return nil, ErrStreamReset
	}
	return frame.DecodeChunk(f.Data)
}

func (s *Stream) readChunkAck() (uint32, error) {
	flags := frame.FlagACK | frame.FlagRST
	f, err := s.read(flags, true)
	if err != nil {
		return 0, err
	}
	if f.IsRST() {
		return 0, ErrStreamReset
	}
	ca, err := frame.DecodeChunkAck(f.Data)
	if err != nil {
		return 0, err
	}
	return uint32(ca), nil
}

func (s *Stream) readRst() error {
	_, err := s.read(frame.FlagRST, true)
	return err
}

func (s *Stream) writeInit(length int) error {
	d := frame.Init(length).Bytes()
	return s.writeFull(frame.FlagSYN, d)
}

func (s *Stream) writeInitAck() error {
	flags := frame.FlagSYN | frame.FlagACK
	return s.writeFull(flags, nil)
}

func (s *Stream) writeChunk(seq uint32, off int, b []byte) (int, error) {
	c := frame.NewChunk(seq, uint32(off), b)
	cb := c.Bytes()
	n, err := s.write(0, cb)
	if err != nil {
		return 0, err
	}
	length := 0
	if b != nil {
		length = len(b)
	}
	delta := len(cb) - length
	return n - delta, nil
}

func (s *Stream) writeChunkAck(seq uint32) error {
	d := frame.ChunkAck(seq).Bytes()
	return s.writeFull(frame.FlagACK, d)
}

func (s *Stream) writeRst() error {
	return s.writeFull(frame.FlagRST, nil)
}

func (s *Stream) writeFin() error {
	return s.writeFull(frame.FlagFIN, nil)
}

func (s *Stream) writeFull(flags uint8, b []byte) error {
	n, err := s.write(flags, b)
	if err != nil {
		return err
	}
	length := 0
	if b != nil {
		length = len(b)
	}
	if n < length {
		return io.ErrShortWrite
	}
	return nil
}

func (s *Stream) read(flags uint8, timeout bool) (*frame.Frame, error) {
	timer := timerPool.Get().(*time.Timer)
	timer.Reset(defaultTimeout)
	for {
		select {
		case f := <-s.readCh:
			if flags == 0 || f.Flags&flags != 0 {
				return &f, nil
			}
		case err := <-s.errorCh:
			return nil, err
		case <-timer.C:
			if timeout {
				return nil, ErrReadTimeout
			}
		case <-s.die:
			return nil, io.EOF
		}
	}
}

func (s *Stream) write(flags uint8, b []byte) (int, error) {
	f := frame.New(flags, s.id, b)
	fb := f.Bytes()
	log.Debugf("[%s] %5s: %s", s.session, "write", f)
	n, err := s.session.Write(fb)
	if err != nil {
		return 0, err
	}
	length := 0
	if b != nil {
		length = len(b)
	}
	delta := len(fb) - length
	return n - delta, nil
}

func (s *Stream) dispatch(f frame.Frame) {
	if f.IsFIN() {
		if err := s.close(true, false); err != nil {
			s.errorCh <- err
		}
	} else {
		s.readCh <- f
	}
}

func (s *Stream) close(detach bool, notify bool) error {
	if err := s.errIfClosed(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if notify {
		if err := s.writeFin(); err != nil {
			return err
		}
	}
	close(s.die)
	if detach {
		s.session.remove(s.id)
	}
	s.closed = true
	return nil
}

func (s *Stream) errIfClosed() error {
	if s.Closed() {
		return ErrStreamClosed
	}
	return nil
}
