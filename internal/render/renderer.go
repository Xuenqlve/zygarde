package render

import (
	"context"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

// Request contains the normalized data required to render runtime artifacts.
type Request struct {
	Prepared runtime.PreparedRuntime
	Contexts []runtime.EnvironmentContext
}

// Renderer generates runtime artifacts from normalized environment contexts.
type Renderer interface {
	Render(ctx context.Context, req Request) (*model.RenderResult, error)
}
