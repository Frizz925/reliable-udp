package mux

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"reliable-udp/protocol"
	"reliable-udp/util"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrStreamClosed      = errors.New("stream closed")
	ErrStreamInterrupted = errors.New("stream interrupted")
)

type StreamConfig struct {
	SessionConfig
	sid protocol.StreamID
}

type Stream struct {
	cfg StreamConfig

	session *Session
	sid     protocol.StreamID

	readState struct {
		sync.Mutex
		buffer     []byte
		readOffset int
		highOffset int
	}

	inbound chan protocol.Frame
	die     chan struct{}
	mu      sync.Mutex

	deadline      atomic.Value
	readDeadline  atomic.Value
	writeDeadline atomic.Value

	closed util.AtomicBool
}

var _ net.Conn = (*Stream)(nil)

func NewStream(session *Session, cfg StreamConfig) *Stream {
	conn := &Stream{
		cfg:     cfg,
		session: session,
		sid:     cfg.sid,
		inbound: make(chan protocol.Frame, InboundBufferSize),
		die:     make(chan struct{}),
	}
	conn.readState.buffer = make([]byte, cfg.bufferSize)
	log.Debugf("Created new stream: %+v", conn)
	return conn
}

func (conn *Stream) LocalAddr() net.Addr {
	return conn.session.LocalAddr()
}

func (conn *Stream) RemoteAddr() net.Addr {
	return conn.session.RemoteAddr()
}

func (conn *Stream) Read(b []byte) (int, error) {
	if conn.closed.Get() {
		return 0, ErrStreamClosed
	}
	conn.readState.Lock()
	defer conn.readState.Unlock()

	blen := len(b)
	// Non-zero high offset indicates that we have previously received stream of data
	if conn.readState.highOffset > 0 {
		// If read offset is less than the high offset then we still have
		// remaining data in the internal buffer to read
		if conn.readState.highOffset > conn.readState.readOffset {
			start := conn.readState.readOffset
			end := conn.readState.highOffset
			length := end - start
			if length > blen {
				length = blen
				end = start + length
			}
			copy(b, conn.readState.buffer[start:end])
			conn.readState.readOffset = end
			return length, nil
		} else {
			// Otherwise we just reset our read state
			conn.readState.highOffset = 0
			conn.readState.readOffset = 0
		}
	}

	// We won't know the content length until we receive the FIN frame
	// Until then the length should be our internal buffer size
	length := len(conn.readState.buffer)
	read := 0
	for read < length {
		f, err := conn.recv(protocol.FrameStreamData, protocol.FrameStreamDataFin)
		if err != nil {
			return 0, err
		}

		sdf, ok := f.(protocol.StreamDataFin)
		if ok {
			// We received our FIN frame, set the known content length
			length = int(sdf.Length)
			// If the length is larger than our internal buffer, then stop receiving stream
			if length > len(conn.readState.buffer) {
				conn.resetRead()
				return 0, bytes.ErrTooLarge
			}
			// Otherwise if our read count is greater or equal than the length, then we're done reading
			if read >= length {
				break
			}
			continue
		}

		// Start processing the received stream data
		sd, ok := f.(protocol.StreamData)
		if !ok {
			continue
		}
		offset, data := int(sd.Offset), sd.Data
		dataLen := len(data)

		// If the high offset is somehow larger than our internal buffer size, then stop receiving stream
		highOffset := offset + dataLen
		if highOffset > length {
			conn.resetRead()
			return 0, bytes.ErrTooLarge
		}

		// Start writing the received data into our internal buffer
		for i := 0; i < dataLen; i++ {
			conn.readState.buffer[offset+i] = data[i]
		}

		conn.send(protocol.NewStreamDataAck(conn.sid, uint32(offset)))
		read += dataLen
	}
	conn.readState.highOffset = read

	// If we somehow receive content larger than the buffer,
	// then write back at most the size of that buffer
	if read > blen {
		read = blen
	}
	copy(b, conn.readState.buffer[:read])
	conn.readState.readOffset = read

	return read, nil
}

func (conn *Stream) Write(b []byte) (int, error) {
	if conn.closed.Get() {
		return 0, ErrStreamClosed
	}

	frameSize := conn.cfg.maxFrameSize
	bufSize := conn.cfg.bufferSize
	if len(b) > bufSize {
		b = b[:bufSize]
	}
	length := len(b)

	written := 0
	ackMap := make(map[int]bool)
	for written < length {
		offset := written
		highOffset := offset + frameSize
		if highOffset > length {
			highOffset = length
		}
		data := b[offset:highOffset]
		conn.send(protocol.NewStreamData(conn.sid, uint32(offset), data))
		ackMap[offset] = false
		written = highOffset
	}
	conn.send(protocol.NewStreamDataFin(conn.sid, uint(written)))

	// Wait for all ACKs to be received
	for len(ackMap) > 0 {
		f, err := conn.recv(protocol.FrameStreamDataAck)
		if err != nil {
			return 0, err
		}
		sda, ok := f.(protocol.StreamDataAck)
		if !ok {
			continue
		}
		offset := int(sda.Offset)
		delete(ackMap, offset)
	}

	return written, nil
}

func (conn *Stream) SetDeadline(t time.Time) error {
	if conn.closed.Get() {
		return ErrStreamClosed
	}
	conn.deadline.Store(t)
	return nil
}

func (conn *Stream) SetReadDeadline(t time.Time) error {
	if conn.closed.Get() {
		return ErrStreamClosed
	}
	conn.readDeadline.Store(t)
	return nil
}

func (conn *Stream) SetWriteDeadline(t time.Time) error {
	if conn.closed.Get() {
		return ErrStreamClosed
	}
	conn.writeDeadline.Store(t)
	return nil
}

func (conn *Stream) Close() error {
	return conn.close(true)
}

func (conn *Stream) String() string {
	return fmt.Sprintf("Stream(StreamID: %d, Session: %+v)", conn.sid, conn.session)
}

func (conn *Stream) send(frame protocol.Frame) {
	conn.session.SendStream(WithDeferCrypto(frame))
}

func (conn *Stream) recv(types ...protocol.FrameType) (protocol.Frame, error) {
	for {
		select {
		case frame := <-conn.inbound:
			if len(types) <= 0 {
				return frame, nil
			}
			for _, ft := range types {
				if ft == frame.Type() {
					return frame, nil
				}
			}
		case <-conn.die:
			return nil, ErrStreamInterrupted
		}
	}
}

func (conn *Stream) streamFrame(ft protocol.FrameType) protocol.StreamFrame {
	return protocol.NewStreamFrame(ft, conn.sid)
}

func (conn *Stream) sendStreamFrame(ft protocol.FrameType) {
	conn.send(conn.streamFrame(ft))
}

func (conn *Stream) resetRead() {
	conn.readState.readOffset = 0
	conn.readState.highOffset = 0
	conn.sendStreamFrame(protocol.FrameStreamReset)
}

func (conn *Stream) dispatch(frame protocol.Frame) {
	conn.inbound <- frame
}

func (conn *Stream) close(detach bool) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.closed.Get() {
		return ErrStreamClosed
	}
	conn.closed.Set(true)
	log.Debugf("Closing stream: %+v", conn)
	conn.sendStreamFrame(protocol.FrameStreamClose)
	close(conn.die)
	if detach {
		conn.session.remove(conn.sid)
	}
	return nil
}
