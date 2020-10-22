package protocol

import (
	"bytes"
	"fmt"
	"io"
)

type Raw struct {
	ft      FrameType
	content []byte
}

func NewRaw(ft FrameType, b []byte) *Raw {
	return &Raw{
		ft:      ft,
		content: b,
	}
}

func ReadRaw(r io.Reader, ft FrameType) (*Raw, error) {
	fn, err := ReadVariableLength(r)
	if err != nil {
		return nil, err
	}
	if fn > MaxPacketSize {
		return nil, bytes.ErrTooLarge
	}
	b, err := ReadBytes(r, int(fn))
	if err != nil {
		return nil, err
	}
	return NewRaw(ft, b), nil
}

func (r *Raw) Type() FrameType {
	return r.ft
}

func (r *Raw) Content() []byte {
	return r.content
}

func (r *Raw) Buffer() io.Reader {
	return bytes.NewBuffer(r.content)
}

func (r *Raw) Serialize(w io.Writer) error {
	fn := uint(len(r.content))
	if fn > MaxPacketSize {
		return bytes.ErrTooLarge
	}
	if err := WriteVariableLength(w, fn); err != nil {
		return err
	}
	return WriteFull(w, r.content)
}

func (r *Raw) String() string {
	return fmt.Sprintf("Raw(Type: %d, Length: %d)", r.ft, len(r.content))
}
