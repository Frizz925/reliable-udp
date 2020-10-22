package client

import (
	"bytes"
	"crypto/cipher"
	"io"
	"net"
	"reliable-udp/protocol"
	"sync"
	"sync/atomic"

	"golang.org/x/crypto/chacha20poly1305"
)

type Client struct {
	conn *net.UDPConn
	aead cipher.AEAD
	cid  protocol.ConnectionID
	seq  uint32
}

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(protocol.EmptyBuffer())
	},
}

func New(conn *net.UDPConn, pk protocol.PrivateKey) (*Client, error) {
	aead, err := chacha20poly1305.New(pk[:])
	if err != nil {
		return nil, err
	}
	cid, err := protocol.GenerateConnectionID()
	if err != nil {
		return nil, err
	}
	return &Client{
		conn: conn,
		aead: aead,
		cid:  cid,
	}, nil
}

func (c *Client) Send(frame protocol.Frame, raddr *net.UDPAddr) error {
	seq := atomic.AddUint32(&c.seq, 1)
	nonce := protocol.Uint32ToNonce(seq)
	cf, err := protocol.EncryptFrame(c.aead, nonce, frame)
	if err != nil {
		return err
	}
	packet := protocol.Packet{
		ConnectionID: c.cid,
		Sequence:     seq,
		Frame:        cf,
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()
	if err := packet.Serialize(buf); err != nil {
		return err
	}
	b := buf.Bytes()
	n, err := c.conn.WriteToUDP(b, raddr)
	if err != nil {
		return err
	}
	if n < len(b) {
		return io.ErrShortWrite
	}
	return nil
}
