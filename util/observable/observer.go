package observable

import "sync"

type Observer struct {
	ob     *Observable
	id     int
	ch     chan interface{}
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

func NewObserver(ob *Observable, id int) *Observer {
	return &Observer{
		ob:   ob,
		id:   id,
		ch:   make(chan interface{}, 1),
		done: make(chan struct{}, 1),
	}
}

func (o *Observer) Next() <-chan interface{} {
	return o.ch
}

func (o *Observer) Done() <-chan struct{} {
	return o.done
}

func (o *Observer) Dispatch(v interface{}) {
	o.ch <- v
}

func (o *Observer) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	close(o.done)
	close(o.ch)
	o.ob.remove(o.id)
	o.closed = true
}
