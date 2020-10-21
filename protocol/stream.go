package protocol

import "io"

type StreamID uint32

func ReadStreamID(r io.Reader) (StreamID, error) {
	sid, err := ReadUint32(r)
	if err != nil {
		return 0, err
	}
	return StreamID(sid), nil
}

func (sid StreamID) Serialize(w io.Writer) error {
	return WriteUint32(w, uint32(sid))
}

type StreamFrame interface {
	Frame
	StreamID() StreamID
}

type BaseStreamFrame struct {
	sid StreamID
	ft  FrameType
}

func (bsf *BaseStreamFrame) Type() FrameType {
	return bsf.ft
}

func (bsf *BaseStreamFrame) Serialize(w io.Writer) error {
	return bsf.sid.Serialize(w)
}
