package protocol

import "io"

type StreamID uint32

func DecodeStreamID(b []byte) (StreamID, error) {
	if len(b) < 4 {
		return 0, io.ErrShortBuffer
	}
	return StreamID(BytesToUint32(b)), nil
}

func (sid StreamID) Bytes() []byte {
	return Uint32ToBytes(uint32(sid))
}
