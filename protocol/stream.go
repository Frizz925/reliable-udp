package protocol

import (
	"fmt"
	"io"
)

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

type StreamFrame struct {
	ft  FrameType
	sid StreamID
}

func ReadStreamFrame(r io.Reader, ft FrameType) (*StreamFrame, error) {
	sid, err := ReadStreamID(r)
	if err != nil {
		return nil, err
	}
	return &StreamFrame{
		ft:  ft,
		sid: sid,
	}, nil
}

func (sf *StreamFrame) Type() FrameType {
	return sf.ft
}

func (sf *StreamFrame) StreamID() StreamID {
	return sf.sid
}

func (sf *StreamFrame) Serialize(w io.Writer) error {
	return sf.sid.Serialize(w)
}

func (sf *StreamFrame) String() string {
	return fmt.Sprintf("StreamFrame(StreamID: %d, Type: %d)", sf.sid, sf.ft)
}
