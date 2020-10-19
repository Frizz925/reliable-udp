package protocol

import (
	"errors"
	"io"
	"math"
)

var ErrVariableLengthOverflow = errors.New("variable length overflow")

const (
	maxUint6  = math.MaxUint8 / 4
	maxUint14 = math.MaxUint16 / 4
	maxUint30 = math.MaxUint32 / 4
	maxUint62 = math.MaxUint64 / 4
)

const (
	flagUint6 byte = iota
	flagUint14
	flagUint30
	flagUint62
)

func EncodeVariableLength(length uint) ([]byte, error) {
	if length > maxUint62 {
		return nil, ErrVariableLengthOverflow
	}
	b := make([]byte, 0)
	flag := flagUint6
	switch {
	case length <= maxUint6:
		b = []byte{byte(length)}
	case length <= maxUint14:
		b = Uint16ToBytes(uint16(length))
		flag = flagUint14
	case length <= maxUint30:
		b = Uint32ToBytes(uint32(length))
		flag = flagUint30
	default:
		b = Uint64ToBytes(uint64(length))
		flag = flagUint62
	}
	b[0] = byte((flag << 6) | (b[0] & maxUint6))
	return b, nil
}

func DecodeVariableLength(b []byte) (uint, error) {
	if len(b) < 1 {
		return 0, io.ErrShortBuffer
	}
	flag := b[0] >> 6
	b[0] = b[0] & maxUint6
	switch flag {
	case flagUint62:
		if len(b) < 8 {
			return 0, io.ErrShortBuffer
		}
		return uint(BytesToUint64(b)), nil
	case flagUint30:
		if len(b) < 4 {
			return 0, io.ErrShortBuffer
		}
		return uint(BytesToUint32(b)), nil
	case flagUint14:
		if len(b) < 2 {
			return 0, io.ErrShortBuffer
		}
		return uint(BytesToUint16(b)), nil
	default:
		return uint(b[0]), nil
	}
}
