package protocol

import (
	"crypto/rand"
	"encoding/binary"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

const (
	KeySize   = chacha20poly1305.KeySize
	NonceSize = chacha20poly1305.NonceSize
)

type (
	PrivateKey [KeySize]byte
	PublicKey  [KeySize]byte
	Nonce      [NonceSize]byte
)

// Reader can be nil to generate new private key from cryptographically secure random generator.
func ReadPrivateKey(r io.Reader) (priv PrivateKey, err error) {
	if r == nil {
		r = rand.Reader
	}
	return priv, ReadFull(r, priv[:])
}

func (priv PrivateKey) PublicKey() (pub PublicKey, err error) {
	return CreatePublicKey(priv[:])
}

func (priv PrivateKey) SharedSecret(pub PublicKey) ([]byte, error) {
	return SharedSecret(priv[:], pub[:])
}

func (priv PrivateKey) Serialize(w io.Writer) error {
	return WriteFull(w, priv[:])
}

func CreatePublicKey(key []byte) (pub PublicKey, err error) {
	b, err := curve25519.X25519(key, curve25519.Basepoint)
	if err != nil {
		return pub, err
	}
	copy(pub[:], b)
	return pub, nil
}

func ReadPublicKey(r io.Reader) (pub PublicKey, err error) {
	return pub, ReadFull(r, pub[:])
}

func (pub PublicKey) SharedSecret(priv PrivateKey) ([]byte, error) {
	return SharedSecret(priv[:], pub[:])
}

func (pub PublicKey) Serialize(w io.Writer) error {
	return WriteFull(w, pub[:])
}

func SharedSecret(k0, k1 []byte) ([]byte, error) {
	return curve25519.X25519(k0, k1)
}

func Uint32ToNonce(v uint32) (n Nonce) {
	// We use little-endian for nonce
	binary.LittleEndian.PutUint32(n[:], v)
	return n
}
