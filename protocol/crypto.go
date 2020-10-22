package protocol

import (
	"bytes"
	"crypto/cipher"
	"io"
)

type Crypto struct {
	ciphertext []byte
	plaintext  []byte
}

func ReadCrypto(r io.Reader, aead cipher.AEAD, nonce Nonce) (*Crypto, error) {
	raw, err := ReadRaw(r, FrameCrypto)
	if err != nil {
		return nil, err
	}
	ciphertext := raw.Content()
	plaintext, err := aead.Open(nil, nonce[:], ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return &Crypto{
		ciphertext: ciphertext,
		plaintext:  plaintext,
	}, nil
}

func EncryptFrame(aead cipher.AEAD, nonce Nonce, f Frame) (*Crypto, error) {
	buf := &bytes.Buffer{}
	if err := WriteFrame(buf, f); err != nil {
		return nil, err
	}
	raw := NewRaw(FrameCrypto, buf.Bytes())
	plaintext := raw.Content()
	ciphertext := aead.Seal(nil, nonce[:], plaintext, nil)
	return &Crypto{
		ciphertext: ciphertext,
		plaintext:  plaintext,
	}, nil
}

func (c *Crypto) Type() FrameType {
	return FrameCrypto
}

func (c *Crypto) Plaintext() []byte {
	return c.plaintext
}

func (c *Crypto) Ciphertext() []byte {
	return c.ciphertext
}

func (c *Crypto) Frame() (Frame, error) {
	buf := bytes.NewBuffer(c.plaintext)
	return ReadFrame(buf)
}

func (c *Crypto) Serialize(w io.Writer) error {
	return NewRaw(FrameCrypto, c.ciphertext).Serialize(w)
}
