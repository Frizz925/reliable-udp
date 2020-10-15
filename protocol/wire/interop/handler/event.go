package handler

import (
	"net"
	"reliable-udp/protocol/frame"
)

type Event struct {
	Frame      *frame.Frame
	RemoteAddr *net.UDPAddr
	Error      error
}

func NewEvent(raddr *net.UDPAddr, err error) Event {
	return Event{
		RemoteAddr: raddr,
		Error:      err,
	}
}
