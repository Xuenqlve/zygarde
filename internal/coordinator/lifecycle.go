package coordinator

import (
	"context"
	"fmt"
	"time"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

// EnvironmentRequest identifies one persisted environment.
type EnvironmentRequest struct {
	EnvironmentID string
}

// CreateResult captures the user-facing create output.
type CreateResult struct {
	EnvironmentID string
	WorkspaceDir  string
	ProjectName   string
	Message       string
}

// Result captures one user-facing lifecycle action result.
type Result struct {
	Message string
}

// Doctor executes one persisted environment diagnostic flow through its runtime driver.
func (c Coordinator) Doctor(ctx context.Context, req EnvironmentRequest) (*Result, error) {
	env, artifact, driver, err := c.loadEnvironmentRuntime(req.EnvironmentID)
	if err != nil {
		return nil, err
	}

	plan, err := driver.PlanLifecycle(ctx, runtime.BuildLifecycleRequest{
		Environment: env,
		Artifact:    artifact,
	})
	if err != nil {
		return nil, err
	}

	result, err := driver.Doctor(ctx, *plan)
	if err != nil {
		return nil, err
	}

	return &Result{
		Message: fmt.Sprintf("environment %s doctor passed: %s", env.ID, result.Message),
	}, nil
}

// Status queries one persisted environment through its runtime driver.
func (c Coordinator) Status(ctx context.Context, req EnvironmentRequest) (*Result, error) {
	env, artifact, driver, err := c.loadEnvironmentRuntime(req.EnvironmentID)
	if err != nil {
		return nil, err
	}

	plan, err := driver.PlanLifecycle(ctx, runtime.BuildLifecycleRequest{
		Environment: env,
		Artifact:    artifact,
	})
	if err != nil {
		return nil, err
	}

	result, err := driver.Status(ctx, *plan)
	if err != nil {
		return nil, err
	}

	env.Status = result.Status
	env.Endpoints = result.Endpoints
	env.UpdatedAt = time.Now()
	if err := c.environments.Save(env); err != nil {
		return nil, err
	}

	return &Result{
		Message: fmt.Sprintf("environment %s status: %s (%s)", env.ID, result.Status, result.Message),
	}, nil
}

// Start starts one persisted environment and records the new status.
func (c Coordinator) Start(ctx context.Context, req EnvironmentRequest) (*Result, error) {
	env, artifact, driver, err := c.loadEnvironmentRuntime(req.EnvironmentID)
	if err != nil {
		return nil, err
	}

	plan, err := driver.PlanLifecycle(ctx, runtime.BuildLifecycleRequest{
		Environment: env,
		Artifact:    artifact,
	})
	if err != nil {
		return nil, err
	}

	result, err := driver.Start(ctx, *plan)
	if err != nil {
		return nil, err
	}

	env.Status = model.EnvironmentStatusRunning
	env.Endpoints = result.Endpoints
	env.UpdatedAt = time.Now()
	if err := c.environments.Save(env); err != nil {
		return nil, err
	}

	return &Result{
		Message: fmt.Sprintf("environment %s started: %s", env.ID, result.Message),
	}, nil
}

// Stop stops one persisted environment and records the new status.
func (c Coordinator) Stop(ctx context.Context, req EnvironmentRequest) (*Result, error) {
	env, artifact, driver, err := c.loadEnvironmentRuntime(req.EnvironmentID)
	if err != nil {
		return nil, err
	}

	plan, err := driver.PlanLifecycle(ctx, runtime.BuildLifecycleRequest{
		Environment: env,
		Artifact:    artifact,
	})
	if err != nil {
		return nil, err
	}

	result, err := driver.Stop(ctx, *plan)
	if err != nil {
		return nil, err
	}

	env.Status = model.EnvironmentStatusStopped
	env.UpdatedAt = time.Now()
	if err := c.environments.Save(env); err != nil {
		return nil, err
	}

	return &Result{
		Message: fmt.Sprintf("environment %s stopped: %s", env.ID, result.Message),
	}, nil
}

// Down stops and removes one persisted environment, then cleans up its local runtime workspace.
func (c Coordinator) Down(ctx context.Context, req EnvironmentRequest) (*Result, error) {
	env, artifact, driver, err := c.loadEnvironmentRuntime(req.EnvironmentID)
	if err != nil {
		return nil, err
	}

	plan, err := driver.PlanLifecycle(ctx, runtime.BuildLifecycleRequest{
		Environment: env,
		Artifact:    artifact,
	})
	if err != nil {
		return nil, err
	}

	destroyResult, err := driver.Destroy(ctx, *plan)
	if err != nil {
		return nil, err
	}

	cleanupResult, err := driver.Cleanup(ctx, *plan)
	if err != nil {
		return nil, err
	}

	env.Status = model.EnvironmentStatusDestroyed
	env.Endpoints = nil
	env.UpdatedAt = time.Now()
	if err := c.environments.Save(env); err != nil {
		return nil, err
	}

	return &Result{
		Message: fmt.Sprintf(
			"environment %s down completed: %s; cleanup: %s",
			env.ID,
			destroyResult.Message,
			cleanupResult.Message,
		),
	}, nil
}

// Destroy is kept as a compatibility alias of Down.
func (c Coordinator) Destroy(ctx context.Context, req EnvironmentRequest) (*Result, error) {
	return c.Down(ctx, req)
}

func (c Coordinator) loadEnvironmentDriver(id string) (model.Environment, runtime.Driver, error) {
	env, err := c.environments.Get(id)
	if err != nil {
		return model.Environment{}, nil, err
	}

	driver, err := c.runtimes.Get(runtime.EnvironmentType(env.RuntimeType))
	if err != nil {
		return model.Environment{}, nil, err
	}

	return env, driver, nil
}

func (c Coordinator) loadEnvironmentRuntime(id string) (model.Environment, runtime.RuntimeArtifact, runtime.Driver, error) {
	env, driver, err := c.loadEnvironmentDriver(id)
	if err != nil {
		return model.Environment{}, runtime.RuntimeArtifact{}, nil, err
	}
	artifact, err := c.environments.GetRuntimeArtifact(id)
	if err != nil {
		return model.Environment{}, runtime.RuntimeArtifact{}, nil, err
	}
	return env, artifact, driver, nil
}
