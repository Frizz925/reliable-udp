package observable

import (
	"sync"
)

type ObserveFunc func(*Observer, interface{})

type DisposeFunc func()

type Observable struct {
	mu        sync.RWMutex
	observers map[int]*Observer
	nextId    int
}

func New() *Observable {
	return &Observable{
		observers: make(map[int]*Observer),
	}
}

func (o *Observable) Observe() *Observer {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.nextId++
	ob := NewObserver(o, o.nextId)
	o.observers[ob.id] = ob
	return ob
}

func (o *Observable) ObserveFunc(handler ObserveFunc) DisposeFunc {
	ob := o.Observe()
	go observe(ob, handler)
	return ob.Close
}

func (o *Observable) Dispatch(i interface{}) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	for _, ob := range o.observers {
		ob.Dispatch(i)
	}
}

func (o *Observable) remove(id int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.observers, id)
}

func observe(ob *Observer, handler ObserveFunc) {
	for {
		select {
		case v := <-ob.Next():
			handler(ob, v)
		case <-ob.Done():
			return
		}
	}
}
