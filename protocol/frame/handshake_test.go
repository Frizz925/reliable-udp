package frame

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandshake(t *testing.T) {
	require := require.New(t)

	// Test decode sanity check
	{
		_, err := DecodeHandshake(make([]byte, 0))
		require.Equal(ErrBufferUnderflow, err)
	}

	// Test encode/decode corectness
	{
		data := "Hello, world!"
		hash := BytesToMD5Hash([]byte(data))

		expected := Handshake{
			StreamID: StreamID(1),
			Length:   uint16(len(data)),
			Reserved: 0,
			Hash:     hash,
			Padding:  HandshakeDefaultPaddingSize,
		}
		actual, err := DecodeHandshake(expected.Bytes())
		require.Nil(err)
		require.Equal(expected.StreamID, actual.StreamID)
		require.Equal(expected.Length, actual.Length)
		require.Equal(expected.Reserved, actual.Reserved)
		require.Equal(expected.Hash, actual.Hash)
		require.Equal(expected.Padding, actual.Padding)
	}
}

func TestHandshakeAck(t *testing.T) {
	require := require.New(t)

	// Test decode sanity check
	{
		_, err := DecodeHandshakeAck(make([]byte, 0))
		require.Equal(ErrBufferUnderflow, err)
	}

	// Test encode/decode corectness
	{
		expected := HandshakeAck{
			StreamID: StreamID(1),
			Size:     65535,
		}
		actual, err := DecodeHandshakeAck(expected.Bytes())
		require.Nil(err)
		require.Equal(expected.StreamID, actual.StreamID)
		require.Equal(expected.Size, actual.Size)
	}
}
