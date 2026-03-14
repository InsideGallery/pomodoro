package module

import "github.com/InsideGallery/pomodoro/internal/event"

// Module is a self-contained feature that reacts to timer events.
type Module interface {
	ID() string
	Init(bus *event.Bus)
	Enabled() bool
}

// Registry holds registered modules and provides lookup.
type Registry struct {
	modules []Module
	bus     *event.Bus
}

func NewRegistry(bus *event.Bus) *Registry {
	return &Registry{bus: bus}
}

func (r *Registry) Register(m Module) {
	m.Init(r.bus)
	r.modules = append(r.modules, m)
}

func (r *Registry) Modules() []Module {
	return r.modules
}

func (r *Registry) ByID(id string) Module { //nolint:ireturn // registry lookup returns interface by design
	for _, m := range r.modules {
		if m.ID() == id {
			return m
		}
	}

	return nil
}
