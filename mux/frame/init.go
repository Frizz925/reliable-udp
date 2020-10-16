package frame

import "io"

const sizeOfInit = 4

type Init uint32

func DecodeInit(b []byte) (Init, error) {
	if len(b) < sizeOfInit {
		return 0, io.ErrShortBuffer
	}
	return Init(BytesToUint32(b)), nil
}

func (i Init) Bytes() []byte {
	return Uint32ToBytes(uint32(i))
}
