package mux

import (
	"fmt"
	"reliable-udp/protocol"
)

// LazyCrypto is a partial frame which carries unencrypted frame.
// Due to its partial nature, its serialize function would result in serialization of the unencrypted frame.
// LazyCrypto is simply a type that helps serializer routine to check whether a frame should be encrypted during the serialization process.
type LazyCrypto struct {
	protocol.Frame
}

func CryptoFrame(frame protocol.Frame) LazyCrypto {
	return LazyCrypto{frame}
}

func (lc LazyCrypto) String() string {
	return fmt.Sprintf("LazyCrypto(%+v)", lc.Frame)
}
