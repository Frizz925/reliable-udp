package interop

import "net"

type UDPConn interface {
	WriteToUDP([]byte, *net.UDPAddr) (int, error)
	ReadFromUDP([]byte) (int, *net.UDPAddr, error)
}
