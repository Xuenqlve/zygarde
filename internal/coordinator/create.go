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
	"github.com/xuenqlve/zygarde/internal/tool"
)

// CreateRequest contains the minimum input for the reserved create-only flow.
type CreateRequest struct {
	BlueprintFile   string
	EnvironmentType runtime.EnvironmentType
}

// UpRequest contains the minimum input for the up flow.
type UpRequest struct {
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

// Create is reserved for the future create-only flow that prepares environment assets without starting runtime.
func (c Coordinator) Create(ctx context.Context, req CreateRequest) (*CreateResult, error) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	normalized, runtimeContexts, driver, err := c.prepareCreateFlow(req.BlueprintFile, req.EnvironmentType)
	if err != nil {
		return nil, err
	}

	var (
		env      model.Environment
		artifact runtime.RuntimeArtifact
		plan     runtime.LifecyclePlan
		canSave  bool
		canPlan  bool
	)

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
	env = prepared.Environment
	artifact = runtime.RuntimeArtifact{
		EnvironmentID: env.ID,
		RuntimeType:   req.EnvironmentType,
		WorkspaceDir:  prepared.Layout.RootDir,
		ProjectName:   prepared.ProjectName,
		PrimaryFile:   prepared.Layout.ComposeFile,
		Files:         prepared.Files,
	}
	plan = runtime.LifecyclePlan{
		Environment:  env,
		WorkspaceDir: prepared.Layout.RootDir,
		ProjectName:  prepared.ProjectName,
		PrimaryFile:  prepared.Layout.ComposeFile,
	}
	canSave = env.ID != ""
	canPlan = plan.WorkspaceDir != ""

	rendered, err := driver.Render(ctx, runtime.RenderRequest{
		Prepared: *prepared,
		Contexts: runtimeContexts,
	})
	if err != nil {
		return nil, c.handleUpFailure(ctx, driver, env, artifact, plan, canSave, canPlan, false, err)
	}
	artifact.PrimaryFile = rendered.PrimaryFile
	plan.PrimaryFile = rendered.PrimaryFile

	applyPlan, err := driver.PlanApply(ctx, runtime.BuildApplyRequest{
		Prepared: *prepared,
		Rendered: *rendered,
		Contexts: runtimeContexts,
	})
	if err != nil {
		return nil, c.handleUpFailure(ctx, driver, env, artifact, plan, canSave, canPlan, false, err)
	}
	artifact.WorkspaceDir = applyPlan.WorkspaceDir
	artifact.ProjectName = applyPlan.ProjectName
	artifact.PrimaryFile = applyPlan.PrimaryFile
	plan = runtime.LifecyclePlan{
		Environment:  env,
		WorkspaceDir: applyPlan.WorkspaceDir,
		ProjectName:  applyPlan.ProjectName,
		PrimaryFile:  applyPlan.PrimaryFile,
	}
	canPlan = plan.WorkspaceDir != ""

	created, err := driver.Create(ctx, *applyPlan)
	if err != nil {
		return nil, c.handleUpFailure(ctx, driver, env, artifact, plan, canSave, canPlan, true, err)
	}

	env.Status = model.EnvironmentStatusStopped
	env.Endpoints = nil
	env.LastError = ""
	env.UpdatedAt = time.Now()
	if err := c.environments.Save(env); err != nil {
		return nil, c.handleUpFailure(ctx, driver, env, artifact, plan, canSave, canPlan, true, fmt.Errorf("save environment: %w", err))
	}
	if err := c.environments.SaveRuntimeArtifact(runtime.RuntimeArtifact{
		EnvironmentID: env.ID,
		RuntimeType:   req.EnvironmentType,
		WorkspaceDir:  applyPlan.WorkspaceDir,
		ProjectName:   applyPlan.ProjectName,
		PrimaryFile:   applyPlan.PrimaryFile,
		Files:         prepared.Files,
	}); err != nil {
		return nil, c.handleUpFailure(ctx, driver, env, artifact, plan, canSave, canPlan, true, fmt.Errorf("save runtime artifact: %w", err))
	}

	return &CreateResult{
		EnvironmentID: env.ID,
		WorkspaceDir:  applyPlan.WorkspaceDir,
		ProjectName:   applyPlan.ProjectName,
		Message: fmt.Sprintf(
			"created environment %s for %s with %d service(s), %d runtime context(s), compose file %s; %s; environment is stopped",
			env.ID,
			normalized.Name,
			len(normalized.Services),
			len(runtimeContexts),
			rendered.PrimaryFile,
			created.Message,
		),
	}, nil
}

// Up loads the blueprint, normalizes services, renders runtime artifacts, applies runtime changes, and records the environment metadata.
func (c Coordinator) Up(ctx context.Context, req UpRequest) (*CreateResult, error) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	normalized, runtimeContexts, driver, err := c.prepareCreateFlow(req.BlueprintFile, req.EnvironmentType)
	if err != nil {
		return nil, err
	}

	var (
		env        model.Environment
		artifact   runtime.RuntimeArtifact
		plan       runtime.LifecyclePlan
		canSave    bool
		canPlan    bool
		canDestroy bool
	)

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
	env = prepared.Environment
	artifact = runtime.RuntimeArtifact{
		EnvironmentID: env.ID,
		RuntimeType:   req.EnvironmentType,
		WorkspaceDir:  prepared.Layout.RootDir,
		ProjectName:   prepared.ProjectName,
		PrimaryFile:   prepared.Layout.ComposeFile,
		Files:         prepared.Files,
	}
	plan = runtime.LifecyclePlan{
		Environment:  env,
		WorkspaceDir: prepared.Layout.RootDir,
		ProjectName:  prepared.ProjectName,
		PrimaryFile:  prepared.Layout.ComposeFile,
	}
	canSave = env.ID != ""
	canPlan = plan.WorkspaceDir != ""

	rendered, err := driver.Render(ctx, runtime.RenderRequest{
		Prepared: *prepared,
		Contexts: runtimeContexts,
	})
	if err != nil {
		return nil, c.handleUpFailure(ctx, driver, env, artifact, plan, canSave, canPlan, false, err)
	}
	artifact.PrimaryFile = rendered.PrimaryFile
	plan.PrimaryFile = rendered.PrimaryFile

	applyPlan, err := driver.PlanApply(ctx, runtime.BuildApplyRequest{
		Prepared: *prepared,
		Rendered: *rendered,
		Contexts: runtimeContexts,
	})
	if err != nil {
		return nil, c.handleUpFailure(ctx, driver, env, artifact, plan, canSave, canPlan, false, err)
	}
	artifact.WorkspaceDir = applyPlan.WorkspaceDir
	artifact.ProjectName = applyPlan.ProjectName
	artifact.PrimaryFile = applyPlan.PrimaryFile
	plan = runtime.LifecyclePlan{
		Environment:  env,
		WorkspaceDir: applyPlan.WorkspaceDir,
		ProjectName:  applyPlan.ProjectName,
		PrimaryFile:  applyPlan.PrimaryFile,
	}
	canPlan = plan.WorkspaceDir != ""
	applied, err := driver.Apply(ctx, *applyPlan)
	if err != nil {
		canDestroy = true
		return nil, c.handleUpFailure(ctx, driver, env, artifact, plan, canSave, canPlan, canDestroy, err)
	}
	canDestroy = true

	env.Status = runtimeStatusFromApplyResult(applied)
	env.Endpoints = applied.Endpoints
	env.UpdatedAt = time.Now()
	if err := c.environments.Save(env); err != nil {
		return nil, c.handleUpFailure(ctx, driver, env, artifact, plan, canSave, canPlan, canDestroy, fmt.Errorf("save environment: %w", err))
	}
	if err := c.environments.SaveRuntimeArtifact(runtime.RuntimeArtifact{
		EnvironmentID: env.ID,
		RuntimeType:   req.EnvironmentType,
		WorkspaceDir:  applyPlan.WorkspaceDir,
		ProjectName:   applyPlan.ProjectName,
		PrimaryFile:   applyPlan.PrimaryFile,
		Files:         prepared.Files,
	}); err != nil {
		return nil, c.handleUpFailure(ctx, driver, env, artifact, plan, canSave, canPlan, canDestroy, fmt.Errorf("save runtime artifact: %w", err))
	}

	return &CreateResult{
		EnvironmentID: env.ID,
		WorkspaceDir:  applyPlan.WorkspaceDir,
		ProjectName:   applyPlan.ProjectName,
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

func (c Coordinator) handleUpFailure(
	ctx context.Context,
	driver runtime.Driver,
	env model.Environment,
	artifact runtime.RuntimeArtifact,
	plan runtime.LifecyclePlan,
	canSave bool,
	canCleanup bool,
	canDestroy bool,
	cause error,
) error {
	lastError := cause.Error()

	if canSave {
		env.Status = model.EnvironmentStatusError
		env.Endpoints = nil
		env.LastError = lastError
		env.UpdatedAt = time.Now()
		if err := c.environments.Save(env); err != nil {
			lastError = appendFailure(lastError, "save environment", err)
		}
	}

	if canSave && artifact.EnvironmentID != "" {
		if err := c.environments.SaveRuntimeArtifact(artifact); err != nil {
			lastError = appendFailure(lastError, "save runtime artifact", err)
		}
	}

	if canDestroy && canCleanup {
		if _, err := driver.Destroy(ctx, plan); err != nil {
			lastError = appendFailure(lastError, "destroy", err)
		}
	}
	if canCleanup {
		if _, err := driver.Cleanup(ctx, plan); err != nil {
			lastError = appendFailure(lastError, "cleanup", err)
		}
	}

	if canSave {
		env.Status = model.EnvironmentStatusError
		env.Endpoints = nil
		env.LastError = lastError
		env.UpdatedAt = time.Now()
		if err := c.environments.Save(env); err != nil {
			lastError = appendFailure(lastError, "save environment", err)
		}
	}

	if env.ID == "" {
		return fmt.Errorf("up failed: %s", lastError)
	}
	return fmt.Errorf("up failed for environment %s: %s", env.ID, lastError)
}

func appendFailure(message, label string, err error) string {
	if err == nil {
		return message
	}
	return message + "; " + label + " failed: " + err.Error()
}

func (c Coordinator) prepareCreateFlow(
	blueprintFile string,
	envType runtime.EnvironmentType,
) (model.Blueprint, []runtime.EnvironmentContext, runtime.Driver, error) {
	loaded, err := c.blueprints.LoadBlueprint(blueprintFile)
	if err != nil {
		return model.Blueprint{}, nil, nil, err
	}

	normalized, err := blueprint.Normalize(loaded, envType)
	if err != nil {
		return model.Blueprint{}, nil, nil, err
	}

	middlewareSet := make(map[string]template.Middleware)
	middlewares := make([]template.Middleware, 0, len(normalized.Services))

	for index, service := range normalized.Services {
		middleware, middlewareErr := template.GetMiddleware(
			template.NewMiddlewareRuntimeKey(service.Middleware, service.Template, envType),
		)
		if middlewareErr != nil {
			return model.Blueprint{}, nil, nil, middlewareErr
		}

		if _, configureErr := middleware.Configure(template.ServiceInput{
			Name:       service.Name,
			Middleware: service.Middleware,
			Template:   service.Template,
			Values:     service.Values,
		}, index+1); configureErr != nil {
			return model.Blueprint{}, nil, nil, configureErr
		}

		key := service.Middleware + "_" + service.Template + "_" + string(envType)
		if _, ok := middlewareSet[key]; !ok {
			middlewareSet[key] = middleware
			middlewares = append(middlewares, middleware)
		}
	}

	runtimeContexts := make([]runtime.EnvironmentContext, 0, len(normalized.Services))
	for _, middleware := range middlewares {
		contexts, buildErr := middleware.BuildRuntimeContexts(envType)
		if buildErr != nil {
			return model.Blueprint{}, nil, nil, buildErr
		}
		runtimeContexts = append(runtimeContexts, contexts...)
	}

	driver, err := c.runtimes.Get(envType)
	if err != nil {
		return model.Blueprint{}, nil, nil, err
	}

	return normalized, runtimeContexts, driver, nil
}
