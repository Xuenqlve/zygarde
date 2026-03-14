package runtime

import (
	"context"

	"github.com/xuenqlve/zygarde/internal/model"
)

// PrepareRequest contains the normalized input for runtime initialization.
type PrepareRequest struct {
	Blueprint model.Blueprint
	Contexts  []EnvironmentContext
}

// PreparedRuntime contains the runtime layout and initialized environment metadata.
type PreparedRuntime struct {
	Environment model.Environment
	Layout      model.RuntimeLayout
}

// RenderRequest contains the input needed to generate runtime artifacts.
type RenderRequest struct {
	Prepared PreparedRuntime
	Contexts []EnvironmentContext
}

// ApplyRequest contains the runtime artifacts to apply to one environment.
type ApplyRequest struct {
	Prepared PreparedRuntime
	Rendered model.RenderResult
}

// OperationResult describes one runtime lifecycle action result.
type OperationResult struct {
	Message   string
	Changed   bool
	Endpoints []model.Endpoint
}

// StatusResult describes one runtime status query result.
type StatusResult struct {
	Status    model.EnvironmentStatus
	Message   string
	Endpoints []model.Endpoint
}

// Driver defines the lifecycle contract each runtime backend must implement.
type Driver interface {
	Type() EnvironmentType
	Prepare(ctx context.Context, req PrepareRequest) (*PreparedRuntime, error)
	Render(ctx context.Context, req RenderRequest) (*model.RenderResult, error)
	Apply(ctx context.Context, req ApplyRequest) (*OperationResult, error)
	Status(ctx context.Context, env model.Environment) (*StatusResult, error)
	Start(ctx context.Context, env model.Environment) (*OperationResult, error)
	Stop(ctx context.Context, env model.Environment) (*OperationResult, error)
	Destroy(ctx context.Context, env model.Environment) (*OperationResult, error)
	Cleanup(ctx context.Context, env model.Environment) (*OperationResult, error)
}
