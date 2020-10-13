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
		expected := Stream{
			StreamID: StreamID(5),
			Sequence: 10,
			Offset:   15,
			Chunk:    []byte(data),
		}
		actual, err := DecodeStream(expected.Bytes())
		require.Nil(err)
		require.Equal(expected.StreamID, actual.StreamID)
		require.Equal(expected.Sequence, actual.Sequence)
		require.Equal(expected.Offset, actual.Offset)
		require.Equal(expected.Chunk, actual.Chunk)
	}
}

func TestStreamAck(t *testing.T) {
	require := require.New(t)

	// Test decode sanity check
	{
		_, err := DecodeStreamAck(make([]byte, 0))
		require.Equal(ErrBufferUnderflow, err)
	}

	// Test encode/decode corectness
	{
		expected := StreamAck{
			StreamID: StreamID(1),
			Sequence: 2,
		}
		actual, err := DecodeStreamAck(expected.Bytes())
		require.Nil(err)
		require.Equal(expected.StreamID, actual.StreamID)
		require.Equal(expected.Sequence, actual.Sequence)
	}
}
