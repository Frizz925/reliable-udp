package protocol

import (
	"io"
	"math/rand"
)

const (
	// Minimum MTU for IPv6 is 1280 bytes, the IPv6 header size is 40 bytes,
	// and UDP header size 8 bytes. From these numbers, the maximum packet
	// size this protocol can support is 1232 bytes.
	MaxPacketSize = 1232
	// u8 Type + u64 Connection ID + u32 Sequence
	PacketHeaderSize = 13
	// u16 Frame Length
	FrameLengthSize = 2
)

type PacketType uint8

const (
	TypeNone uint8 = iota
	TypeHandshake
	TypeStreamOpen
	TypeStreamReset
	TypeStreamDataInit
	TypeStreamData
	TypeStreamDataAck
	TypeStreamClose
	TypeTerminate
)

type Packet struct {
	Type         uint8
	ConnectionID uint64
	Sequence     uint32
	Frame
}

func GenerateConnectionID() uint64 {
	b := make([]byte, 8)
	rand.Read(b)
	return BytesToUint64(b)
}

func Decode(b []byte) (*Packet, error) {
	if len(b) < PacketHeaderSize {
		return nil, io.ErrShortBuffer
	}
	p := &Packet{
		Type:         uint8(b[0]),
		ConnectionID: BytesToUint64(b[1:9]),
		Sequence:     BytesToUint32(b[9:13]),
	}
	if len(b) < PacketHeaderSize+FrameLengthSize {
		return p, nil
	}
	frameLen := int(BytesToUint16(b[13:15]))
	frameBytes := b[15:]
	if len(frameBytes) < frameLen {
		return nil, io.ErrShortBuffer
	}
	frameBytes = frameBytes[:frameLen]
	switch p.Type {
	case TypeHandshake:
	case TypeStreamOpen:
		fallthrough
	case TypeStreamReset:
		fallthrough
	case TypeStreamDataAck:
		fallthrough
	case TypeStreamClose:
		frame, err := DecodeStreamID(frameBytes)
		if err != nil {
			return nil, err
		}
		p.Frame = frame
	default:
		p.Frame = Raw(frameBytes)
	}
	return p, nil
}
