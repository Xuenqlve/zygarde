package template

import (
	"fmt"

	"github.com/xuenqlve/zygarde/internal/model"
)

// NormalizeServices resolves specs, normalizes user inputs, and validates uniqueness.
func NormalizeServices(registry Registry, inputs []ServiceInput) ([]model.BlueprintService, error) {
	services := make([]model.BlueprintService, 0, len(inputs))
	seenNames := make(map[string]struct{}, len(inputs))

	for i, input := range inputs {
		spec, err := ResolveSpec(registry, input)
		if err != nil {
			return nil, err
		}

		service, err := spec.Normalize(input, i+1)
		if err != nil {
			return nil, err
		}

		if service.Middleware == "" {
			service.Middleware = spec.Middleware()
		}
		if service.Template == "" {
			service.Template = spec.Template()
		}
		if service.Name == "" {
			return nil, ErrServiceNameRequired
		}
		if service.Template == "" {
			return nil, ErrServiceTemplateRequired
		}

		if _, exists := seenNames[service.Name]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateServiceName, service.Name)
		}
		seenNames[service.Name] = struct{}{}

		if err := spec.Validate(service); err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	return services, nil
}

// ResolveSpec finds a concrete service spec for one service input.
func ResolveSpec(registry Registry, input ServiceInput) (ServiceSpec, error) {
	if input.Middleware == "" {
		return nil, ErrMiddlewareRequired
	}

	if input.Template == "" {
		spec, ok := registry.Default(input.Middleware)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrDefaultSpecNotFound, input.Middleware)
		}
		return spec, nil
	}

	spec, ok := registry.Get(input.Middleware, input.Template)
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrSpecNotFound, input.Middleware, input.Template)
	}
	return spec, nil
}

// DefaultServiceName builds the fallback name for one service input.
func DefaultServiceName(middleware string, index int) string {
	return fmt.Sprintf("%s-%d", middleware, index)
}

// MergeValues returns a copy of defaults with user values applied on top.
func MergeValues(defaults, overrides map[string]any) map[string]any {
	merged := make(map[string]any, len(defaults)+len(overrides))
	for key, value := range defaults {
		merged[key] = value
	}
	for key, value := range overrides {
		merged[key] = value
	}
	return merged
}
