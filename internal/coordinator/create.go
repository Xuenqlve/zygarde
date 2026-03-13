package coordinator

import (
	"context"

	"github.com/xuenqlve/zygarde/internal/blueprint"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	"github.com/xuenqlve/zygarde/internal/store"
	"github.com/xuenqlve/zygarde/internal/template"
)

// CreateRequest contains the minimum input for the create flow.
type CreateRequest struct {
	BlueprintFile   string
	EnvironmentType runtime.EnvironmentType
}

// CreateResult captures the create flow output before render/deploy.
type CreateResult struct {
	Blueprint       model.Blueprint
	EnvironmentType runtime.EnvironmentType
	Middlewares     []template.Middleware
}

// Coordinator orchestrates the create flow.
type Coordinator struct {
	blueprints store.BlueprintStore
}

// New creates a coordinator instance.
func New(blueprints store.BlueprintStore) Coordinator {
	return Coordinator{blueprints: blueprints}
}

// Create loads the blueprint, normalizes services, and calls middleware Configure.
func (c Coordinator) Create(_ context.Context, req CreateRequest) (*CreateResult, error) {
	loaded, err := c.blueprints.LoadBlueprint(req.BlueprintFile)
	if err != nil {
		return nil, err
	}

	normalized, err := blueprint.Normalize(loaded, req.EnvironmentType)
	if err != nil {
		return nil, err
	}

	middlewareSet := make(map[string]template.Middleware)
	middlewares := make([]template.Middleware, 0, len(normalized.Services))

	for _, service := range normalized.Services {
		middleware, err := template.GetMiddleware(
			template.NewMiddlewareRuntimeKey(service.Middleware, service.Template, req.EnvironmentType),
		)
		if err != nil {
			return nil, err
		}

		if _, err := middleware.Configure(template.ServiceInput{
			Name:       service.Name,
			Middleware: service.Middleware,
			Template:   service.Template,
			Values:     service.Values,
		}, 0); err != nil {
			return nil, err
		}

		key := service.Middleware + "_" + service.Template + "_" + string(req.EnvironmentType)
		if _, ok := middlewareSet[key]; !ok {
			middlewareSet[key] = middleware
			middlewares = append(middlewares, middleware)
		}
	}

	return &CreateResult{
		Blueprint:       normalized,
		EnvironmentType: req.EnvironmentType,
		Middlewares:     middlewares,
	}, nil
}
