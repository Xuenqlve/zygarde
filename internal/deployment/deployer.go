package deployment

import (
	"context"

	"github.com/xuenqlve/zygarde/internal/runtime"
)

// Executor defines runtime-backed deployment lifecycle operations.
type Executor interface {
	Apply(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error)
	Status(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.StatusResult, error)
	Doctor(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error)
	Start(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error)
	Stop(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error)
	Destroy(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error)
	Cleanup(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error)
}
