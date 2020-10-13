package frame

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFin(t *testing.T) {
	require := require.New(t)

	// Test decode sanity check
	{
		_, err := DecodeFin(make([]byte, 0))
		require.Equal(ErrBufferUnderflow, err)
	}

	// Test encode/decode corectness
	{
		expected := Fin{StreamID: 1}
		actual, err := DecodeFin(expected.Bytes())
		require.Nil(err)
		require.Equal(expected.StreamID, actual.StreamID)
	}
}
