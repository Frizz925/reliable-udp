package protocol

import (
	"bytes"
	"crypto/cipher"
	"fmt"
	"io"
)

type Crypto []byte

func ReadCrypto(r io.Reader) (Crypto, error) {
	raw, err := ReadRaw(r)
	if err != nil {
		return nil, err
	}
	return Crypto(raw), nil
}

func (Crypto) Type() FrameType {
	return FrameCrypto
}

func (c Crypto) Serialize(w io.Writer) error {
	return WriteRaw(w, c[:])
}

func (c Crypto) String() string {
	return fmt.Sprintf("Crypto(Length: %d)", len(c))
}

type CryptoFacade struct {
	aead      cipher.AEAD
	frameSize int
}

func NewCryptoFacade(aead cipher.AEAD, maxFrameSize int) *CryptoFacade {
	frameSize := maxFrameSize - aead.NonceSize() - aead.Overhead()
	return &CryptoFacade{
		aead:      aead,
		frameSize: frameSize,
	}
}

func (cf *CryptoFacade) Decrypt(crypto Crypto, nonce Nonce) (Frame, error) {
	plaintext, err := cf.aead.Open(nil, nonce[:], crypto[:], nil)
	if err != nil {
		return nil, err
	}
	raw, err := ReadRaw(bytes.NewReader(plaintext))
	if err != nil {
		return nil, err
	}
	return ReadFrame(bytes.NewReader(raw))
}

func (cf *CryptoFacade) Encrypt(frame Frame, nonce Nonce) (Crypto, error) {
	frameBuf := &bytes.Buffer{}
	if err := WriteFrame(frameBuf, frame); err != nil {
		return nil, err
	}
	rawBuf := &bytes.Buffer{}
	if err := WriteRaw(rawBuf, frameBuf.Bytes()); err != nil {
		return nil, err
	}
	paddingSize := cf.frameSize - rawBuf.Len()
	padding := make([]byte, paddingSize)
	rawBuf.Write(padding)
	ciphertext := cf.aead.Seal(nil, nonce[:], rawBuf.Bytes(), nil)
	return Crypto(ciphertext), nil
}
