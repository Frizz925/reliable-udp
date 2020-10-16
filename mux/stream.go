package mux

import (
	"errors"
	"io"
	"reliable-udp/mux/frame"
	"sync"
)

const streamReadBacklog = 128

var (
	ErrStreamInterrupted = errors.New("stream interrupted")
	ErrStreamClosed      = errors.New("stream closed")
)

type Stream struct {
	peer *Peer
	id   uint32

	mu   sync.RWMutex
	rwmu sync.Mutex

	readCh chan Packet
	die    chan struct{}

	closed bool
}

func NewStream(peer *Peer, id uint32) *Stream {
	return &Stream{
		peer:   peer,
		id:     id,
		readCh: make(chan Packet, streamReadBacklog),
		die:    make(chan struct{}),
	}
}

func (s *Stream) Read(b []byte) (int, error) {
	if err := s.reset(); err != nil {
		return 0, err
	}
	s.rwmu.Lock()
	defer s.rwmu.Unlock()
	size, err := s.readInit()
	if err != nil {
		return 0, err
	}
	defer s.writeFin()
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
	defer s.writeFin()
	if err := s.readInitAck(); err != nil {
		return 0, err
	}
	seq := uint32(0)
	written := 0
	for written < len(b) {
		seq++
		n, err := s.writeChunk(seq, written, b[written:])
		if err != nil {
			return 0, err
		}
		written += n
	}
	return written, nil
}

func (s *Stream) Closed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

func (s *Stream) Close() error {
	return s.close(true)
}

func (s *Stream) readInit() (int, error) {
	for {
		f, err := s.read(frame.FlagSYN)
		if err != nil {
			return 0, err
		}
		seq, err := frame.DecodeInit(f.Data)
		return int(seq), err
	}
}

func (s *Stream) readInitAck() error {
	for {
		_, err := s.read(frame.FlagSYN | frame.FlagACK)
		return err
	}
}

func (s *Stream) readChunk() (*frame.Chunk, error) {
	f, err := s.read(0)
	if err != nil {
		return nil, err
	}
	return frame.DecodeChunk(f.Data)
}

func (s *Stream) readChunkAck(seq uint32) error {
	d := frame.ChunkAck(seq).Bytes()
	return s.writeFull(frame.FlagACK, d)
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
	d := frame.NewChunk(seq, uint32(off), b).Bytes()
	return s.write(0, d)
}

func (s *Stream) writeChunkAck(seq uint32) error {
	d := frame.ChunkAck(seq).Bytes()
	return s.writeFull(frame.FlagACK, d)
}

func (s *Stream) writeFin() error {
	return s.writeFull(frame.FlagFIN, nil)
}

func (s *Stream) read(flags uint8) (*frame.Frame, error) {
	for {
		if err := s.errIfClosed(); err != nil {
			return nil, err
		}
		select {
		case pa := <-s.readCh:
			if pa.Flags&frame.FlagFIN != 0 {
				return nil, ErrStreamInterrupted
			}
			if flags == 0 || pa.Flags&flags != 0 {
				return &pa.Frame, nil
			}
		case <-s.die:
			return nil, ErrStreamInterrupted
		}
	}
}

func (s *Stream) write(flags uint8, b []byte) (int, error) {
	if err := s.errIfClosed(); err != nil {
		return 0, err
	}
	f := frame.New(flags, s.id, b)
	fb := f.Bytes()
	n, err := s.peer.write(fb)
	if err != nil {
		return 0, err
	}
	delta := len(fb) - f.Length()
	return n - delta, nil
}

func (s *Stream) reset() error {
	if err := s.errIfClosed(); err != nil {
		return err
	}
	for {
		select {
		case <-s.readCh:
		case <-s.die:
			return ErrStreamInterrupted
		default:
			return nil
		}
	}
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

func (s *Stream) dispatch(pa Packet) {
	if !s.Closed() {
		s.readCh <- pa
	}
}

func (s *Stream) close(remove bool) error {
	if err := s.errIfClosed(); err != nil {
		return err
	}
	if remove {
		s.peer.remove(s.id)
	}
	s.mu.Lock()
	close(s.die)
	s.peer = nil
	s.closed = true
	s.mu.Unlock()
	return nil
}

func (s *Stream) errIfClosed() error {
	if s.Closed() {
		return ErrStreamClosed
	}
	return nil
}
