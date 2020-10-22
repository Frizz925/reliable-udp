package protocol

import (
	"io"
)

const MaxKeySize = 32

type Handshake struct {
	BufferSize uint
	PublicKey  PublicKey
}

func ReadHandshake(r io.Reader) (*Handshake, error) {
	bufferSize, err := ReadVariableLength(r)
	if err != nil {
		return nil, err
	}
	pubKey, err := ReadPublicKey(r)
	if err != nil {
		return nil, err
	}
	return &Handshake{
		BufferSize: bufferSize,
		PublicKey:  pubKey,
	}, nil
}

func (h *Handshake) Serialize(w io.Writer) error {
	if err := WriteVariableLength(w, h.BufferSize); err != nil {
		return err
	}
	return h.PublicKey.Serialize(w)
}
