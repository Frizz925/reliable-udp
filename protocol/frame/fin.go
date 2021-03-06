package frame

// FIN frame informs the peer that we're closing either a stream or the underlying connection
// as a whole. FIN frame may not arrive to the peer and thus the peer may not know that we
// have closed the stream or connection. It's up to peer's application to handle this condition
// such as by implementing a timeout behavior when the peer fails to receive ACK frames in
// a certain period of time.
//
// Non-zero value indicates the stream to close,
// while zero value indicates that the whole underlying connection to close.
type Fin struct {
	StreamID
}

func DecodeFin(b []byte) (*Fin, error) {
	sid, err := DecodeStreamID(b)
	if err != nil {
		return nil, err
	}
	return &Fin{sid}, nil
}

func (Fin) Type() FrameType {
	return FinType
}
