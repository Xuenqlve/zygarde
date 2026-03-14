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
	Message string
}

// Result captures one user-facing lifecycle action result.
type Result struct {
	Message string
}

// Status queries one persisted environment through its runtime driver.
func (c Coordinator) Status(ctx context.Context, req EnvironmentRequest) (*Result, error) {
	env, driver, err := c.loadEnvironmentDriver(req.EnvironmentID)
	if err != nil {
		return nil, err
	}

	result, err := driver.Status(ctx, env)
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
	env, driver, err := c.loadEnvironmentDriver(req.EnvironmentID)
	if err != nil {
		return nil, err
	}

	result, err := driver.Start(ctx, env)
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
	env, driver, err := c.loadEnvironmentDriver(req.EnvironmentID)
	if err != nil {
		return nil, err
	}

	result, err := driver.Stop(ctx, env)
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

// Destroy destroys one persisted environment, then cleans up its local runtime workspace.
func (c Coordinator) Destroy(ctx context.Context, req EnvironmentRequest) (*Result, error) {
	env, driver, err := c.loadEnvironmentDriver(req.EnvironmentID)
	if err != nil {
		return nil, err
	}

	destroyResult, err := driver.Destroy(ctx, env)
	if err != nil {
		return nil, err
	}

	cleanupResult, err := driver.Cleanup(ctx, env)
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
			"environment %s destroyed: %s; cleanup: %s",
			env.ID,
			destroyResult.Message,
			cleanupResult.Message,
		),
	}, nil
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
