package handler

import "reliable-udp/util/observable"

type baseHandler struct {
	ch   chan Event
	err  chan error
	done chan struct{}
}

func newBaseHandler() *baseHandler {
	return &baseHandler{
		ch:   make(chan Event, 1),
		err:  make(chan error, 1),
		done: make(chan struct{}),
	}
}

func (h *baseHandler) OnDispose(o *observable.Observer) {
	close(h.err)
	close(h.done)
}
