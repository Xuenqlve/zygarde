package deployment

import (
	"context"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

// Executor defines runtime-backed deployment lifecycle operations.
type Executor interface {
	Apply(ctx context.Context, env model.Environment, rendered model.RenderResult) (*runtime.OperationResult, error)
	Status(ctx context.Context, env model.Environment) (*runtime.StatusResult, error)
	Start(ctx context.Context, env model.Environment) (*runtime.OperationResult, error)
	Stop(ctx context.Context, env model.Environment) (*runtime.OperationResult, error)
	Destroy(ctx context.Context, env model.Environment) (*runtime.OperationResult, error)
	Cleanup(ctx context.Context, env model.Environment) (*runtime.OperationResult, error)
}
