package coordinator

import (
	"context"
	"fmt"
	"time"

	"github.com/xuenqlve/zygarde/internal/blueprint"
	"github.com/xuenqlve/zygarde/internal/environment"
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

// Coordinator orchestrates the create flow.
type Coordinator struct {
	blueprints   store.BlueprintStore
	environments environment.Store
	runtimes     runtime.Registry
}

// New creates a coordinator instance.
func New(blueprints store.BlueprintStore, environments environment.Store, runtimes runtime.Registry) Coordinator {
	return Coordinator{
		blueprints:   blueprints,
		environments: environments,
		runtimes:     runtimes,
	}
}

// Create loads the blueprint, normalizes services, renders runtime artifacts, and records the environment metadata.
func (c Coordinator) Create(ctx context.Context, req CreateRequest) (*CreateResult, error) {
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

	for index, service := range normalized.Services {
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
		}, index+1); err != nil {
			return nil, err
		}

		key := service.Middleware + "_" + service.Template + "_" + string(req.EnvironmentType)
		if _, ok := middlewareSet[key]; !ok {
			middlewareSet[key] = middleware
			middlewares = append(middlewares, middleware)
		}
	}

	runtimeContexts := make([]runtime.EnvironmentContext, 0, len(normalized.Services))
	for _, middleware := range middlewares {
		contexts, err := middleware.BuildRuntimeContexts(req.EnvironmentType)
		if err != nil {
			return nil, err
		}
		runtimeContexts = append(runtimeContexts, contexts...)
	}

	driver, err := c.runtimes.Get(req.EnvironmentType)
	if err != nil {
		return nil, err
	}

	prepared, err := driver.Prepare(ctx, runtime.PrepareRequest{
		Input: runtime.PrepareInput{
			BlueprintName:    normalized.Name,
			BlueprintVersion: normalized.Version,
			RuntimeType:      req.EnvironmentType,
			RequestedName:    normalized.Name,
			ProjectName:      normalized.Runtime.ProjectName,
			WorkspaceRoot:    ".zygarde/environments",
		},
		Contexts: runtimeContexts,
	})
	if err != nil {
		return nil, err
	}

	rendered, err := driver.Render(ctx, runtime.RenderRequest{
		Prepared: *prepared,
		Contexts: runtimeContexts,
	})
	if err != nil {
		return nil, err
	}

	applyPlan, err := driver.PlanApply(ctx, runtime.BuildApplyRequest{
		Prepared: *prepared,
		Rendered: *rendered,
		Contexts: runtimeContexts,
	})
	if err != nil {
		return nil, err
	}
	applied, err := driver.Apply(ctx, *applyPlan)
	if err != nil {
		return nil, err
	}

	env := prepared.Environment
	env.Status = runtimeStatusFromApplyResult(applied)
	env.Endpoints = applied.Endpoints
	env.UpdatedAt = time.Now()
	if err := c.environments.Save(env); err != nil {
		return nil, err
	}
	if err := c.environments.SaveRuntimeArtifact(runtime.RuntimeArtifact{
		EnvironmentID: env.ID,
		RuntimeType:   req.EnvironmentType,
		WorkspaceDir:  applyPlan.WorkspaceDir,
		ProjectName:   applyPlan.ProjectName,
		PrimaryFile:   applyPlan.PrimaryFile,
		Files:         prepared.Files,
	}); err != nil {
		return nil, err
	}

	return &CreateResult{
		Message: fmt.Sprintf(
			"created environment %s for %s with %d service(s), %d runtime context(s), compose file %s, apply result: %s",
			env.ID,
			normalized.Name,
			len(normalized.Services),
			len(runtimeContexts),
			rendered.PrimaryFile,
			applied.Message,
		),
	}, nil
}

func runtimeStatusFromApplyResult(result *runtime.OperationResult) model.EnvironmentStatus {
	if result != nil {
		return model.EnvironmentStatusRunning
	}
	return model.EnvironmentStatusError
}
