package protocol

import (
	"bytes"
	"crypto/cipher"
	"fmt"
	"io"
)

type Crypto struct {
	frame      Frame
	ciphertext []byte
	plaintext  []byte
}

func ReadCrypto(r io.Reader, aead cipher.AEAD, nonce Nonce) (*Crypto, error) {
	raw, err := ReadRaw(r, FrameCrypto)
	if err != nil {
		return nil, err
	}
	return DecryptFrame(aead, nonce, raw)
}

func DecryptFrame(aead cipher.AEAD, nonce Nonce, raw *Raw) (*Crypto, error) {
	ciphertext := raw.Content()
	plaintext, err := aead.Open(nil, nonce[:], ciphertext, nil)
	if err != nil {
		return nil, err
	}
	frame, err := ReadFrame(bytes.NewBuffer(plaintext))
	if err != nil {
		return nil, err
	}
	return &Crypto{
		frame:      frame,
		ciphertext: ciphertext,
		plaintext:  plaintext,
	}, nil
}

func EncryptFrame(aead cipher.AEAD, nonce Nonce, frame Frame) (*Crypto, error) {
	buf := &bytes.Buffer{}
	if err := WriteFrame(buf, frame); err != nil {
		return nil, err
	}
	raw := NewRaw(FrameCrypto, buf.Bytes())
	plaintext := raw.Content()
	ciphertext := aead.Seal(nil, nonce[:], plaintext, nil)
	return &Crypto{
		frame:      frame,
		ciphertext: ciphertext,
		plaintext:  plaintext,
	}, nil
}

func (c *Crypto) Type() FrameType {
	return FrameCrypto
}

func (c *Crypto) Frame() Frame {
	return c.frame
}

func (c *Crypto) Plaintext() []byte {
	return c.plaintext
}

func (c *Crypto) Ciphertext() []byte {
	return c.ciphertext
}

func (c *Crypto) Serialize(w io.Writer) error {
	return NewRaw(FrameCrypto, c.ciphertext).Serialize(w)
}

func (c *Crypto) String() string {
	return fmt.Sprintf("Crypto(%s)", c.frame)
}
