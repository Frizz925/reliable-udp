package frame

import (
	"bytes"
	"errors"
)

const (
	// FrameType uint8 + Stream ID uint16 + Length uint16
	FrameBaseSize = 5
	// Maximum size of a single frame.
	FrameMaxSize = 1468
	// Maximum size of a single frame's data, already excluding the headers.
	FrameDataMaxSize = FrameMaxSize - FrameBaseSize
)

var (
	ErrBufferUnderflow  = errors.New("buffer underflow")
	ErrFrameTypeUnknown = errors.New("unknown frame type")
	ErrFrameDataMissing = errors.New("missing frame data")
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

type dataDecoderFunc func([]byte) (Data, error)

var dataDecoders = map[FrameType]dataDecoderFunc{
	FinType: func(b []byte) (Data, error) {
		return Fin(BytesToUint16(b)), nil
	},
	HandshakeType: func(b []byte) (Data, error) {
		return DecodeHandshake(b)
	},
	HandshakeAckType: func(b []byte) (Data, error) {
		return HandshakeAck(b[0]), nil
	},
	StreamType: func(b []byte) (Data, error) {
		return DecodeStream(b)
	},
	StreamAckType: func(b []byte) (Data, error) {
		return StreamAck(BytesToUint16(b)), nil
	},
}

// Frame headers consist of frame type, stream ID, and data length.
type Frame struct {
	StreamID uint16
	Data
}

func Decode(b []byte) (*Frame, error) {
	if len(b) < FrameBaseSize {
		return nil, ErrBufferUnderflow
	}
	ft := FrameType(b[0])
	sid := BytesToUint16(b[1:3])
	length := int(BytesToUint16(b[3:5]))
	raw := b[5:]
	if len(raw) < length {
		return nil, ErrBufferUnderflow
	}
	raw = raw[:length]
	data, err := DecodeData(ft, raw)
	if err != nil {
		return nil, err
	}
	return &Frame{
		StreamID: sid,
		Data:     data,
	}, nil
}

func (f *Frame) Length() int {
	if f.Data != nil {
		return len(f.Data.Bytes())
	}
	return 0
}

func (f *Frame) Type() FrameType {
	if f.Data == nil {
		return UnknownType
	}
	return f.Data.Type()
}

func (f *Frame) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(f.Data.Type()))
	buf.Write(Uint16ToBytes(f.StreamID))
	if f.Data != nil {
		raw := f.Data.Bytes()
		buf.Write(Uint16ToBytes(uint16(len(raw))))
		buf.Write(raw)
	} else {
		buf.Write(Uint16ToBytes(0))
	}
	return buf.Bytes()
}

func DecodeData(ft FrameType, b []byte) (Data, error) {
	if decode, ok := dataDecoders[ft]; ok {
		return decode(b)
	}
	return nil, ErrFrameTypeUnknown
}
