package observable

import (
	"sync"
)

type Observer struct {
	ob       *Observable
	id       int
	handlers []Handler
	mu       sync.RWMutex
	disposed bool
}

func NewObserver(ob *Observable, id int) *Observer {
	return &Observer{
		ob:       ob,
		id:       id,
		handlers: make([]Handler, 0),
	}
}

func (o *Observer) HandleFunc(each OnEach, dispose OnDispose) {
	o.Handle(&handlerImpl{each, dispose})
}

func (o *Observer) Handle(handler Handler) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.handlers = append(o.handlers, handler)
}

func (o *Observer) Dispose() {
	o.dispose(true)
}

func (o *Observer) dispatch(v interface{}) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	wg := &sync.WaitGroup{}
	wg.Add(len(o.handlers))
	for _, handler := range o.handlers {
		go o.eachHandler(wg, handler, v)
	}
	wg.Wait()
}

func (o *Observer) dispose(remove bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.disposed {
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(o.handlers))
	for _, handler := range o.handlers {
		go o.disposeHandler(wg, handler)
	}
	wg.Wait()
	if remove {
		o.ob.remove(o.id)
	}
	// Remove reference to avoid memory leaks
	o.ob = nil
	o.handlers = nil
	o.disposed = true
}

func (o *Observer) eachHandler(wg *sync.WaitGroup, h Handler, v interface{}) {
	defer wg.Done()
	h.OnEach(o, v)
}

func (o *Observer) disposeHandler(wg *sync.WaitGroup, h Handler) {
	defer wg.Done()
	h.OnDispose(o)
}
