package observable

type OnEach func(o *Observer, v interface{})

type OnDispose func(o *Observer)

type Handler interface {
	OnEach(o *Observer, v interface{})
	OnDispose(o *Observer)
}

type handlerImpl struct {
	each    OnEach
	dispose OnDispose
}

func (h *handlerImpl) OnEach(o *Observer, v interface{}) {
	if h.each != nil {
		h.each(o, v)
	}
}

func (h *handlerImpl) OnDispose(o *Observer) {
	if h.dispose != nil {
		h.dispose(o)
	}
}
