package frame

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStream(t *testing.T) {
	require := require.New(t)
	data := "Hello, world!"

	// Test decode sanity check
	{
		_, err := DecodeStream(make([]byte, 0))
		require.Equal(ErrBufferUnderflow, err)
	}

	// Test encode/decode corectness
	{
		expected := &Stream{
			Sequence: 5,
			Offset:   10,
			Chunk:    []byte(data),
		}
		actual, err := DecodeStream(expected.Bytes())
		require.Nil(err)
		require.Equal(expected.Sequence, actual.Sequence)
		require.Equal(expected.Offset, actual.Offset)
		require.Equal(expected.Chunk, actual.Chunk)
	}
}
