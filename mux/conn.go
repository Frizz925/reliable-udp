package mux

import "net"

type UDPConn interface {
	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	WriteToUDP(b []byte, raddr *net.UDPAddr) (int, error)
}
