package frame

import "io"

const sizeOfChunkAck = 4

type ChunkAck uint32

func DecodeChunkAck(b []byte) (ChunkAck, error) {
	if len(b) < sizeOfChunkAck {
		return 0, io.ErrShortBuffer
	}
	return ChunkAck(BytesToUint32(b)), nil
}

func (i ChunkAck) Bytes() []byte {
	return Uint32ToBytes(uint32(i))
}
