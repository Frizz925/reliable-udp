package observable

import (
	"sync"
)

type DisposeFunc func()

type Observable struct {
	mu        sync.RWMutex
	observers map[int]*Observer
	nextId    int
	disposed  bool
}

func New() *Observable {
	return &Observable{
		observers: make(map[int]*Observer),
	}
}

func (o *Observable) Observe() *Observer {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.disposed {
		return nil
	}
	o.nextId++
	ob := NewObserver(o, o.nextId)
	o.observers[ob.id] = ob
	return ob
}

func (o *Observable) Dispatch(i interface{}) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.disposed {
		return
	}
	for _, ob := range o.observers {
		ob.dispatch(i)
	}
}

func (o *Observable) Dispose() {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, ob := range o.observers {
		ob.dispose(false)
	}
	o.observers = nil
}

func (o *Observable) remove(id int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.observers, id)
}
