package protocol

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"
)

func TestCrypto(t *testing.T) {
	require := require.New(t)
	randGen := rand.New(rand.NewSource(0))
	content := []byte("Hello, world!")
	expected := NewRaw(FrameRaw, content)
	maxFrameSize := 512
	cryptoFrameSize := maxFrameSize - 128

	key, err := ReadPrivateKey(randGen)
	require.Nil(err)
	aead, err := chacha20poly1305.New(key[:])
	require.Nil(err)
	facade := NewCryptoFacade(aead, maxFrameSize)

	nonce := Uint32ToNonce(1)

	cout, err := facade.Encrypt(expected, nonce)
	require.Nil(err)
	buf := &bytes.Buffer{}
	require.Nil(cout.Serialize(buf))
	require.GreaterOrEqual(buf.Len(), cryptoFrameSize, "Crypto frame should have padding")

	cin, err := ReadCrypto(buf)
	require.Nil(err)
	actual, err := facade.Decrypt(cin, nonce)
	require.Nil(err)
	require.Equal(expected, actual)
}
