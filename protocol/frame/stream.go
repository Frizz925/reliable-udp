package frame

import "bytes"

const (
	// Sequence uint16 + Offset uint16 + Length uint16
	StreamBaseSize = 6
	// Maximum size for the data chunk in a single frame.
	StreamChunkMaxSize = FrameDataMaxSize - StreamBaseSize
)

// Stream frames are chunks of data being sent over the wire, providing stream of data.
type Stream struct {
	// The sequence of the stream packet on the wire.
	Sequence uint16
	// The data chunk offset in a stream.
	Offset uint16
	// The data chunk itself.
	Chunk []byte
}

func DecodeStream(b []byte) (*Stream, error) {
	if len(b) < StreamBaseSize {
		return nil, ErrBufferUnderflow
	}
	chunk := b[6:]
	length := int(BytesToUint16(b[4:6]))
	if len(chunk) < length {
		return nil, ErrBufferUnderflow
	}
	return &Stream{
		Sequence: BytesToUint16(b[:2]),
		Offset:   BytesToUint16(b[2:4]),
		Chunk:    chunk,
	}, nil
}

func (s *Stream) Length() int {
	if s.Chunk != nil {
		return len(s.Chunk)
	}
	return 0
}

func (*Stream) Type() FrameType {
	return StreamType
}

func (s *Stream) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(Uint16ToBytes(s.Sequence))
	buf.Write(Uint16ToBytes(s.Offset))
	if s.Chunk != nil {
		clen := uint16(s.Length())
		buf.Write(Uint16ToBytes(clen))
		buf.Write(s.Chunk)
	} else {
		buf.Write(Uint16ToBytes(0))
	}
	return buf.Bytes()
}

// Stream ACK frames are ACK frames to inform the peer that the we peer have successfully
// received the chunk packet. It indicates which packet sequence that we have received.
type StreamAck uint16

func (StreamAck) Type() FrameType {
	return StreamAckType
}

func (sa StreamAck) Bytes() []byte {
	return Uint16ToBytes(uint16(sa))
}
