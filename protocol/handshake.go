package protocol

import "io"

type Handshake struct {
	BufferSize uint
}

func ReadHandshake(r io.Reader) (*Handshake, error) {
	bufferSize, err := ReadVariableLength(r)
	if err != nil {
		return nil, err
	}
	return &Handshake{
		BufferSize: bufferSize,
	}, nil
}

func (h *Handshake) Serialize(w io.Writer) error {
	return WriteVariableLength(w, h.BufferSize)
}
