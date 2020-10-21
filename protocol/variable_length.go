package protocol

import (
	"bytes"
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

func ReadVariableLength(r io.Reader) (uint, error) {
	firstByte, err := ReadByte(r)
	if err != nil {
		return 0, err
	}
	flag := firstByte >> 6
	firstByte &= maxUint6

	var copyLen int64
	switch flag {
	case flagUint62:
		copyLen = 7
	case flagUint30:
		copyLen = 3
	case flagUint14:
		copyLen = 1
	default:
		return uint(firstByte), nil
	}

	dst := bytes.NewBuffer([]byte{firstByte})
	if _, err := io.CopyN(dst, r, copyLen); err != nil {
		return 0, err
	}
	return BytesToUint(dst.Bytes()), nil
}

func WriteVariableLength(w io.Writer, length uint) error {
	if length > maxUint62 {
		return ErrVariableLengthOverflow
	}
	var b []byte
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
	return WriteFull(w, b)
}
