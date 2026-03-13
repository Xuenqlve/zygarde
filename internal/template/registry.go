package template

import "fmt"

type specKey struct {
	middleware string
	template   string
}

// MemoryRegistry stores service specs in memory.
type MemoryRegistry struct {
	specs    map[specKey]ServiceSpec
	defaults map[string]ServiceSpec
}

// NewRegistry creates an empty registry implementation.
func NewRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		specs:    make(map[specKey]ServiceSpec),
		defaults: make(map[string]ServiceSpec),
	}
}

// Register adds a service spec to the registry.
func (r *MemoryRegistry) Register(spec ServiceSpec) error {
	key := specKey{
		middleware: spec.Middleware(),
		template:   spec.Template(),
	}
	if _, exists := r.specs[key]; exists {
		return fmt.Errorf("%w: %s/%s", ErrSpecAlreadyRegistered, key.middleware, key.template)
	}
	r.specs[key] = spec
	if spec.IsDefault() {
		if _, exists := r.defaults[key.middleware]; exists {
			return fmt.Errorf("%w: %s", ErrDefaultSpecAlreadyRegistered, key.middleware)
		}
		r.defaults[key.middleware] = spec
	}
	return nil
}

// Get returns a spec by middleware and template.
func (r *MemoryRegistry) Get(middleware, template string) (ServiceSpec, bool) {
	spec, ok := r.specs[specKey{
		middleware: middleware,
		template:   template,
	}]
	return spec, ok
}

// Default returns the default spec for one middleware.
func (r *MemoryRegistry) Default(middleware string) (ServiceSpec, bool) {
	spec, ok := r.defaults[middleware]
	return spec, ok
}

// List returns all registered specs.
func (r *MemoryRegistry) List() []ServiceSpec {
	specs := make([]ServiceSpec, 0, len(r.specs))
	for _, spec := range r.specs {
		specs = append(specs, spec)
	}
	return specs
}
