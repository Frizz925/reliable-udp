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

func ReadCrypto(r io.Reader, aead cipher.AEAD, nonce Nonce) (c Crypto, err error) {
	raw, err := ReadRaw(r, FrameCrypto)
	if err != nil {
		return c, err
	}
	return DecryptFrame(aead, nonce, raw)
}

func DecryptFrame(aead cipher.AEAD, nonce Nonce, raw Raw) (c Crypto, err error) {
	c.ciphertext = raw.Content()
	c.plaintext, err = aead.Open(nil, nonce[:], c.ciphertext, nil)
	if err != nil {
		return c, err
	}
	c.frame, err = ReadFrame(bytes.NewReader(c.plaintext))
	if err != nil {
		return c, err
	}
	return c, nil
}

func EncryptFrame(aead cipher.AEAD, nonce Nonce, frame Frame) (c Crypto, err error) {
	c.frame = frame
	buf := &bytes.Buffer{}
	if err := WriteFrame(buf, frame); err != nil {
		return c, err
	}
	raw := NewRaw(FrameCrypto, buf.Bytes())
	c.plaintext = raw.Content()
	c.ciphertext = aead.Seal(nil, nonce[:], c.plaintext, nil)
	return c, err
}

func (c Crypto) Type() FrameType {
	return FrameCrypto
}

func (c Crypto) Frame() Frame {
	return c.frame
}

func (c Crypto) Plaintext() []byte {
	return c.plaintext
}

func (c Crypto) Ciphertext() []byte {
	return c.ciphertext
}

func (c Crypto) Serialize(w io.Writer) error {
	return NewRaw(FrameCrypto, c.ciphertext).Serialize(w)
}

func (c Crypto) String() string {
	return fmt.Sprintf("Crypto(%s)", c.frame)
}
