package protocol

import (
	"bytes"
	"io"
)

const (
	// Minimum MTU for IPv6 is 1280 bytes, the IPv6 header size is 40 bytes,
	// and UDP header size 8 bytes. From these numbers, the maximum packet
	// size this protocol can support is 1232 bytes.
	MaxPacketSize = 1232
	// u8 Type + u64 Connection ID + u32 Sequence
	PacketHeaderSize = 13
)

type Packet struct {
	ConnectionID
	Sequence uint32
	Type     PacketType
	Frame
}

func Deserialize(r io.Reader) (p Packet, err error) {
	p.ConnectionID, err = ReadConnectionID(r)
	if err != nil {
		return
	}
	p.Sequence, err = ReadUint32(r)
	if err != nil {
		return
	}
	p.Type, err = ReadPacketType(r)
	if err != nil {
		return
	}

	ft, err := ReadFrameType(r)
	if err != nil {
		// If no frame type is read then return immediately
		if err == io.EOF {
			return p, nil
		} else {
			return p, err
		}
	}
	fn, err := ReadVariableLength(r)
	if err != nil {
		return p, err
	}
	if fn > MaxPacketSize {
		return p, bytes.ErrTooLarge
	}

	switch ft {
	case FrameStreamOpen:
		fallthrough
	case FrameStreamReset:
		fallthrough
	case FrameStreamDataInit:
		fallthrough
	case FrameStreamDataAck:
		fallthrough
	case FrameStreamClose:
		sid, err := ReadStreamID(r)
		if err != nil {
			return p, err
		}
		p.Frame = &BaseStreamFrame{
			sid: sid,
			ft:  ft,
		}
	default:
		content := make([]byte, fn)
		_, err := io.ReadFull(r, content)
		if err != nil {
			return p, err
		}
		p.Frame = &Raw{
			ft:      ft,
			content: content,
		}
	}

	return p, nil
}

func (p Packet) Serialize(w io.Writer) error {
	if err := p.ConnectionID.Serialize(w); err != nil {
		return err
	}
	if err := WriteUint32(w, p.Sequence); err != nil {
		return err
	}
	if err := p.Type.Serialize(w); err != nil {
		return err
	}
	if p.Frame == nil {
		return nil
	}

	if err := p.Frame.Type().Serialize(w); err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	if err := p.Frame.Serialize(buf); err != nil {
		return err
	}
	if err := WriteVariableLength(w, uint(buf.Len())); err != nil {
		return err
	}
	_, err := io.Copy(w, buf)
	return err
}
