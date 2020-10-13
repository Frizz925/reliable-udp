package wire

import (
	"net"
	"reliable-udp/protocol/frame"
	"sync"
)

// Interop is a thin wrapper around a UDP connection.
// It is responsible for sending and receiving packets
// of data based on the defined protocols.
type Interop struct {
	*net.UDPConn
	mu sync.Mutex
}

func NewInterop(conn *net.UDPConn) *Interop {
	return &Interop{UDPConn: conn}
}

func (i *Interop) Send(ft frame.Type, id uint16, data []byte) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	(frame.Frame{
		Type:     ft,
		StreamID: id,
		Raw:      data,
	})
}

func (i *Interop) SendAck()

func (i *Interop) SendPacket()

func (i *Interop) CloseStream(id int) error {

}

func (i *Interop) Close() error {
	i.WriteToUDP()
}
