package mux

import (
	"net"
	"reliable-udp/mux/frame"
)

const packetMaxSize = 1472

type Packet struct {
	frame.Frame
	RemoteAddr *net.UDPAddr
}

func NewPacket(f frame.Frame, raddr *net.UDPAddr) Packet {
	return Packet{
		Frame:      f,
		RemoteAddr: raddr,
	}
}
