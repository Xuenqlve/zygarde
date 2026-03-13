package app

import (
	"context"

	_ "github.com/xuenqlve/zygarde/pkg/register"

	"github.com/xuenqlve/zygarde/internal/config"
	"github.com/xuenqlve/zygarde/internal/coordinator"
	"github.com/xuenqlve/zygarde/internal/runtime"
	"github.com/xuenqlve/zygarde/internal/store"
)

// App wires the create flow dependencies together.
type App struct {
	cfg         config.Config
	coordinator coordinator.Coordinator
}

// New creates an application instance with the default local dependencies.
func New() (*App, error) {
	return &App{
		cfg:         config.Default(),
		coordinator: coordinator.New(store.NewFileBlueprintStore()),
	}, nil
}

// Create runs the first half of the create flow.
func (a *App) Create(ctx context.Context, blueprintFile string, envType runtime.EnvironmentType) (*coordinator.CreateResult, error) {
	if envType == "" {
		envType = a.cfg.DefaultEnvironmentType
	}

	return a.coordinator.Create(ctx, coordinator.CreateRequest{
		BlueprintFile:   blueprintFile,
		EnvironmentType: envType,
	})
}
