package runtime

import "fmt"

// Registry resolves runtime drivers by environment type.
type Registry struct {
	drivers map[EnvironmentType]Driver
}

// NewRegistry constructs a runtime registry with the provided drivers.
func NewRegistry(drivers ...Driver) (Registry, error) {
	registry := Registry{
		drivers: make(map[EnvironmentType]Driver, len(drivers)),
	}
	for _, driver := range drivers {
		if err := registry.Register(driver); err != nil {
			return Registry{}, err
		}
	}
	return registry, nil
}

// Register adds one runtime driver to the registry.
func (r *Registry) Register(driver Driver) error {
	if driver == nil {
		return fmt.Errorf("runtime driver is required")
	}
	if r.drivers == nil {
		r.drivers = make(map[EnvironmentType]Driver)
	}
	if _, exists := r.drivers[driver.Type()]; exists {
		return fmt.Errorf("runtime driver already registered: %s", driver.Type())
	}
	r.drivers[driver.Type()] = driver
	return nil
}

// Get resolves one runtime driver.
func (r *Registry) Get(envType EnvironmentType) (Driver, error) {
	driver, ok := r.drivers[envType]
	if !ok {
		return nil, fmt.Errorf("runtime driver not found: %s", envType)
	}
	return driver, nil
}
