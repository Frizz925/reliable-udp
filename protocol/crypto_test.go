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
	expected := NewRaw(FrameRaw, []byte("Hello, world!"))

	privKey, err := ReadPrivateKey(randGen)
	require.Nil(err)
	aead, err := chacha20poly1305.New(privKey[:])
	require.Nil(err)
	nonce := Uint32ToNonce(1)

	cout, err := EncryptFrame(aead, nonce, expected)
	require.Nil(err)
	buf := &bytes.Buffer{}
	require.Nil(cout.Serialize(buf))

	cin, err := ReadCrypto(buf, aead, nonce)
	require.Nil(err)
	require.Equal(cout.ciphertext, cin.ciphertext)
	require.Equal(cout.plaintext, cin.plaintext)

	actual := cin.Frame()
	require.Equal(expected, actual)
}
