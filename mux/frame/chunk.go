package frame

import (
	"bytes"
	"io"
)

const (
	sizeOfSequence     = 4
	sizeOfOffset       = 4
	sizeOfChunkHeaders = sizeOfSequence + sizeOfOffset + sizeOfLength
)

type Chunk struct {
	Sequence uint32
	Offset   uint32
	Data     []byte
}

func NewChunk(seq uint32, off uint32, data []byte) Chunk {
	return Chunk{
		Sequence: seq,
		Offset:   off,
		Data:     data,
	}
}

func DecodeChunk(b []byte) (*Chunk, error) {
	if len(b) < sizeOfChunkHeaders {
		return nil, io.ErrShortBuffer
	}
	dlen := int(BytesToUint32(b[8:12]))
	data := b[12:]
	if len(data) < dlen {
		return nil, io.ErrShortBuffer
	}
	data = b[:dlen]
	return &Chunk{
		Sequence: BytesToUint32(b[:4]),
		Offset:   BytesToUint32(b[4:8]),
		Data:     data,
	}, nil
}

func (c Chunk) Length() int {
	return len(c.Data)
}

func (c Chunk) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(Uint32ToBytes(c.Sequence))
	buf.Write(Uint32ToBytes(c.Offset))
	buf.Write(Uint32ToBytes(uint32(len(c.Data))))
	buf.Write(c.Data)
	return buf.Bytes()
}
