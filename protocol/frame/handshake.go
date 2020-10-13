package frame

import (
	"bytes"
)

const (
	// StreamID + Length uint16 + Reserved uint8 + MD5 Hash (128-bit)
	HandshakeBaseSize = StreamIDSize + 19
	// StreamID + Size uint16
	HandshakeAckBaseSize = StreamIDSize + 2
	// The default padding size for the handshake.
	HandshakeDefaultPaddingSize = FrameDataMaxSize - HandshakeBaseSize
)

// Handshake frame is the first frame to send to a peer to start the information exchange
type Handshake struct {
	StreamID
	// How much data the peer should receive in a single stream.
	Length uint16
	// Reserved for hash related information.
	Reserved uint8
	// Hash for data integrity check after the peer has received all the data chunks.
	Hash []byte
	// The received length of the padding. It may not equal to the default padding size.
	// Padding is used for peer to determine the size of a single packet it could receive.
	// The peer is expected to return the size back to us by sending the handshake ACK frame.
	Padding int
}

func DecodeHandshake(b []byte) (*Handshake, error) {
	length := len(b)
	if length < HandshakeBaseSize {
		return nil, ErrBufferUnderflow
	}
	sid, err := DecodeStreamID(b)
	if err != nil {
		return nil, err
	}
	return &Handshake{
		StreamID: sid,
		Length:   BytesToUint16(b[2:]),
		Reserved: b[4],
		Hash:     b[5:HandshakeBaseSize],
		Padding:  length - HandshakeBaseSize,
	}, nil
}

func (Handshake) Type() FrameType {
	return HandshakeType
}

func (h Handshake) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(h.StreamID.Bytes())
	buf.Write(Uint16ToBytes(h.Length))
	buf.WriteByte(h.Reserved)
	buf.Write(h.Hash)
	buf.Write(make([]byte, HandshakeDefaultPaddingSize))
	return buf.Bytes()
}

// Handshake ACK frame is the first frame to send back to the peer.
type HandshakeAck struct {
	StreamID
	// The size of data we can receive in a single frame.
	Size uint16
}

func DecodeHandshakeAck(b []byte) (*HandshakeAck, error) {
	if len(b) < HandshakeAckBaseSize {
		return nil, ErrBufferUnderflow
	}
	sid, err := DecodeStreamID(b)
	if err != nil {
		return nil, err
	}
	size := BytesToUint16(b[2:])
	return &HandshakeAck{sid, size}, nil
}

func (HandshakeAck) Type() FrameType {
	return HandshakeAckType
}

func (ha HandshakeAck) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(ha.StreamID.Bytes())
	buf.Write(Uint16ToBytes(ha.Size))
	return buf.Bytes()
}
