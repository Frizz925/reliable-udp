package frame

import "errors"

const (
	// Type uint8 + Stream ID uint16 + Length uint16
	FrameBaseSize = 5
	// Maximum size of a single frame.
	FrameMaxSize = 1468
	// Maximum size of a single frame's data, already excluding the headers.
	FrameDataMaxSize = FrameMaxSize - FrameBaseSize
)

var (
	ErrBufferUnderflow  = errors.New("buffer underflow")
	ErrUnknownFrameType = errors.New("unknown frame type")
)

type FrameType uint8

const (
	// Unknown frame doesn't exactly get sent out in the wire. However, it simply indicates
	// that the received frame type is either unknown or just not yet set.
	UnknownType FrameType = iota
	FinType
	HandshakeType
	HandshakeAckType
	StreamType
	StreamAckType
)

type Frame struct {
	Type     FrameType
	StreamID uint16
	Raw      []byte
}

func Decode(b []byte) (*Frame, error) {
	if len(b) < FrameBaseSize {
		return nil, ErrBufferUnderflow
	}
	raw := b[5:]
	length := BytesToUint16(b[3:5])
	if len(raw) < int(length) {
		return nil, ErrBufferUnderflow
	}
	return &Frame{
		Type:     FrameType(b[0]),
		StreamID: BytesToUint16(b[1:3]),
		Raw:      raw,
	}, nil
}

func (f Frame) Data() (interface{}, error) {
	switch f.Type {
	case FinType:
		return Fin(BytesToUint16(f.Raw)), nil
	case HandshakeType:
		return DecodeHandshake(f.Raw)
	case HandshakeAckType:
		return HandshakeAck(f.Raw[0]), nil
	case StreamType:
		return DecodeStream(f.Raw)
	case StreamAckType:
		return StreamAck(BytesToUint16(f.Raw)), nil
	}
	return nil, ErrUnknownFrameType
}
