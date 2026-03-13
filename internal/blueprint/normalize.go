package blueprint

import (
	"fmt"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	"github.com/xuenqlve/zygarde/internal/template"
)

// Normalize fills service defaults required before coordinator Configure calls.
func Normalize(blueprint model.Blueprint, envType runtime.EnvironmentType) (model.Blueprint, error) {
	normalized := blueprint
	if normalized.Services == nil {
		normalized.Services = []model.BlueprintService{}
	}

	seenNames := make(map[string]struct{}, len(normalized.Services))
	for i := range normalized.Services {
		service := normalized.Services[i]
		if service.Middleware == "" {
			return model.Blueprint{}, fmt.Errorf("service[%d] middleware is required", i)
		}
		if service.Template == "" {
			middleware, err := template.GetDefaultMiddleware(service.Middleware, envType)
			if err != nil {
				return model.Blueprint{}, err
			}
			service.Template = middleware.Template()
		}
		if service.Name == "" {
			service.Name = template.DefaultServiceName(service.Middleware, i+1)
		}
		if service.Values == nil {
			service.Values = map[string]any{}
		}
		if _, exists := seenNames[service.Name]; exists {
			return model.Blueprint{}, fmt.Errorf("duplicate service name: %s", service.Name)
		}
		seenNames[service.Name] = struct{}{}
		normalized.Services[i] = service
	}

	return normalized, nil
}
