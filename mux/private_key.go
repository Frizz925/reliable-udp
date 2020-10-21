package mux

import (
	"math/rand"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

const KeySize = chacha20poly1305.KeySize

type PrivateKey [KeySize]byte

func GeneratePrivateKey() (PrivateKey, error) {
	b := make([]byte, KeySize)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return PrivateKey(b), nil
}

func (pk PrivateKey) PublicKey() ([]byte, error) {
	return curve25519.X25519(pk, curve25519.Basepoint)
}
