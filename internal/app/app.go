package app

import (
	"context"

	_ "github.com/xuenqlve/zygarde/pkg/register"

	"github.com/xuenqlve/zygarde/internal/config"
	"github.com/xuenqlve/zygarde/internal/coordinator"
	"github.com/xuenqlve/zygarde/internal/environment"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
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
	runtimes, err := runtime.NewRegistry(
		runtimecompose.NewDriver("", nil, nil),
	)
	if err != nil {
		return nil, err
	}

	return &App{
		cfg:         config.Default(),
		coordinator: coordinator.New(
			store.NewFileBlueprintStore(),
			environment.NewFileStore(".zygarde/environments"),
			runtimes,
		),
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

// Status queries one created environment by id.
func (a *App) Status(ctx context.Context, environmentID string) (*coordinator.Result, error) {
	return a.coordinator.Status(ctx, coordinator.EnvironmentRequest{EnvironmentID: environmentID})
}

// Start starts one created environment by id.
func (a *App) Start(ctx context.Context, environmentID string) (*coordinator.Result, error) {
	return a.coordinator.Start(ctx, coordinator.EnvironmentRequest{EnvironmentID: environmentID})
}

// Stop stops one created environment by id.
func (a *App) Stop(ctx context.Context, environmentID string) (*coordinator.Result, error) {
	return a.coordinator.Stop(ctx, coordinator.EnvironmentRequest{EnvironmentID: environmentID})
}

// Destroy destroys one created environment by id.
func (a *App) Destroy(ctx context.Context, environmentID string) (*coordinator.Result, error) {
	return a.coordinator.Destroy(ctx, coordinator.EnvironmentRequest{EnvironmentID: environmentID})
}
