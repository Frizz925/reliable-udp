package protocol

import "io"

type Frame interface {
	Type() FrameType
	Serialize(w io.Writer) error
}
