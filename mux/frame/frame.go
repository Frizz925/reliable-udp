package frame

import (
	"bytes"
	"fmt"
	"io"
)

const (
	FlagSYN uint8 = 1 << iota
	FlagACK
	FlagRST
	FlagFIN
)

const (
	sizeOfFlags    = 1
	sizeOfStreamID = 4
	sizeOfLength   = 4

	sizeOfHeaders = sizeOfFlags + sizeOfStreamID
)

type Frame struct {
	Flags    uint8
	StreamID uint32
	Data     []byte
}

func New(flags uint8, sid uint32, data []byte) Frame {
	return Frame{
		Flags:    flags,
		StreamID: sid,
		Data:     data,
	}
}

func Decode(b []byte) (*Frame, error) {
	if len(b) < sizeOfHeaders {
		return nil, io.ErrShortBuffer
	}
	data := make([]byte, 0)
	if len(b) >= sizeOfHeaders+sizeOfLength {
		chlen := int(BytesToUint32(b[5:9]))
		chunk := b[9:]
		if len(chunk) < chlen {
			return nil, io.ErrShortBuffer
		}
		data = chunk[:chlen]
	}
	return &Frame{
		Flags:    b[0],
		StreamID: BytesToUint32(b[1:5]),
		Data:     data,
	}, nil
}

func (f Frame) IsSYN() bool {
	return f.Flags&FlagSYN != 0
}

func (f Frame) IsACK() bool {
	return f.Flags&FlagACK != 0
}

func (f Frame) IsFIN() bool {
	return f.Flags&FlagFIN != 0
}

func (f Frame) IsRST() bool {
	return f.Flags&FlagRST != 0
}

func (f Frame) Length() int {
	if f.Data != nil {
		return len(f.Data)
	}
	return 0
}

func (f Frame) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteByte(byte(f.Flags))
	buf.Write(Uint32ToBytes(f.StreamID))
	if dlen := f.Length(); dlen > 0 {
		buf.Write(Uint32ToBytes(uint32(dlen)))
		buf.Write(f.Data)
	}
	return buf.Bytes()
}

func (f Frame) String() string {
	return fmt.Sprintf(
		"Flags: %d, StreamID: %d, Length: %d",
		f.Flags, f.StreamID, f.Length(),
	)
}
