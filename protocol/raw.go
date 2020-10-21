package protocol

import "io"

type Raw struct {
	ft      FrameType
	content []byte
}

func (r *Raw) Type() FrameType {
	return r.ft
}

func (r *Raw) Serialize(w io.Writer) error {
	return WriteFull(w, r.content)
}
