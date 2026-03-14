package event

import "sync"

type Handler func(Event)

type Bus struct {
	mu       sync.RWMutex
	handlers map[Type][]Handler
}

func NewBus() *Bus {
	return &Bus{
		handlers: make(map[Type][]Handler),
	}
}

func (b *Bus) Subscribe(t Type, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[t] = append(b.handlers[t], h)
}

func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[e.Type]))
	copy(handlers, b.handlers[e.Type])
	b.mu.RUnlock()

	for _, h := range handlers {
		h(e)
	}
}
