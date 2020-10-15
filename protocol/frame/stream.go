package frame

import "bytes"

const (
	// uint16
	StreamIDSize = 2
	// StreamID + Sequence uint16 + Offset uint16 + Length uint16
	StreamBaseSize = StreamIDSize + 6
	// StreamID + Sequence uint16
	StreamAckBaseSize = StreamIDSize + 2
	// Maximum size for the data chunk in a single frame.
	StreamChunkMaxSize = FrameDataMaxSize - StreamBaseSize
)

// Stream ID is used for multiplexing purposes between streams in a single connection.
type StreamID uint16

func DecodeStreamID(b []byte) (StreamID, error) {
	if len(b) < StreamIDSize {
		return 0, ErrBufferUnderflow
	}
	return StreamID(BytesToUint16(b)), nil
}

func (sid StreamID) Int() int {
	return int(sid)
}

func (sid StreamID) Uint16() uint16 {
	return uint16(sid)
}

func (sid StreamID) Bytes() []byte {
	return Uint16ToBytes(uint16(sid))
}

// Stream frames are chunks of data being sent over the wire, providing stream of data.
type Stream struct {
	StreamID
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
	sid, err := DecodeStreamID(b)
	if err != nil {
		return nil, err
	}
	seq := BytesToUint16(b[2:])
	off := BytesToUint16(b[4:])
	length := int(BytesToUint16(b[6:]))
	chunk := b[8:]
	if len(chunk) < length {
		return nil, ErrBufferUnderflow
	}
	return &Stream{sid, seq, off, chunk}, nil
}

func (s Stream) Length() int {
	if s.Chunk != nil {
		return len(s.Chunk)
	}
	return 0
}

func (Stream) Type() FrameType {
	return StreamType
}

func (s Stream) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(s.StreamID.Bytes())
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

// Stream ACK frames are ACK frames to inform the peer that the we peer have successfully received the chunk packet.
type StreamAck struct {
	StreamID
	// Indicates which packet sequence that we have received.
	Sequence uint16
}

func DecodeStreamAck(b []byte) (*StreamAck, error) {
	if len(b) < StreamAckBaseSize {
		return nil, ErrBufferUnderflow
	}
	sid, err := DecodeStreamID(b)
	if err != nil {
		return nil, err
	}
	seq := BytesToUint16(b[2:])
	return &StreamAck{sid, seq}, nil
}

func (StreamAck) Type() FrameType {
	return StreamAckType
}

func (sa StreamAck) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(sa.StreamID.Bytes())
	buf.Write(Uint16ToBytes(sa.Sequence))
	return buf.Bytes()
}
