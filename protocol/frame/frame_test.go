package frame

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type unknownData struct{}

func (unknownData) Type() FrameType {
	return UnknownType
}

func (unknownData) Bytes() []byte {
	return make([]byte, 0)
}

func TestFrame(t *testing.T) {
	require := require.New(t)

	// Test decode sanity check
	{
		_, err := Decode(make([]byte, 0))
		require.Equal(ErrBufferUnderflow, err)
	}

	// Test decode on nil packet data
	{
		f := Frame{}
		_, err := Decode(f.Bytes())
		require.Equal(ErrFrameTypeUnknown, err)
	}

	// Test decode on unknown packet type
	{
		f := Frame{unknownData{}}
		_, err := Decode(f.Bytes())
		require.Equal(ErrFrameTypeUnknown, err)
	}

	// Test encode/decode corectness
	{
		expected := Stream{
			StreamID: StreamID(1),
			Sequence: 1,
			Offset:   0,
			Chunk:    []byte("Hello, world!"),
		}

		d, err := Decode(Encode(expected))
		require.Nil(err)
		actual, ok := d.Data.(*Stream)
		require.True(ok)

		require.Equal(expected.StreamID, actual.StreamID)
		require.Equal(expected.Sequence, actual.Sequence)
		require.Equal(expected.Offset, actual.Offset)
		require.Equal(expected.Chunk, actual.Chunk)
	}
}
