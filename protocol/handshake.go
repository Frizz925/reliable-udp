package protocol

import (
	"bytes"
	"fmt"
	"io"
)

type Handshake struct {
	FrameType
	BufferSize uint
	PublicKey  PublicKey
	// This field is only filled when reading from byte stream.
	// This works by counting handshake the size of the padding.
	MaxFrameSize int
}

func ReadHandshake(r io.Reader, ft FrameType) (hs Handshake, err error) {
	hs.FrameType = ft
	hs.BufferSize, err = ReadVarInt(r)
	if err != nil {
		return
	}
	hs.PublicKey, err = ReadPublicKey(r)
	if err != nil {
		return
	}
	// Read the rest of the padding to determine the maximum frame size
	b := make([]byte, MaxPacketSize)
	hs.MaxFrameSize, err = r.Read(b)
	return hs, err
}

func (hs Handshake) Type() FrameType {
	return hs.FrameType
}

func (hs Handshake) Serialize(w io.Writer) error {
	if err := WriteVarInt(w, hs.BufferSize); err != nil {
		return err
	}
	if err := hs.PublicKey.Serialize(w); err != nil {
		return err
	}
	buf, ok := w.(*bytes.Buffer)
	if !ok {
		return nil
	}
	paddingSize := MaxPacketSize - buf.Len()
	return WriteFull(w, make([]byte, paddingSize))
}

func (hs Handshake) String() string {
	return fmt.Sprintf(
		"Handshake(Type: %d, BufferSize: %d, MaxFrameSize: %d, PublicKey: %s)",
		hs.FrameType, hs.BufferSize, hs.MaxFrameSize, hs.PublicKey,
	)
}
