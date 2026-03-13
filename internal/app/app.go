package app

import (
	"context"
	"sync"

	"github.com/xuenqlve/zygarde/internal/config"
	"github.com/xuenqlve/zygarde/internal/coordinator"
	"github.com/xuenqlve/zygarde/internal/runtime"
	"github.com/xuenqlve/zygarde/internal/store"
	"github.com/xuenqlve/zygarde/pkg/mysql"
)

var registerOnce sync.Once
var registerErr error

// App wires the create flow dependencies together.
type App struct {
	cfg         config.Config
	coordinator coordinator.Coordinator
}

// New creates an application instance with the default local dependencies.
func New() (*App, error) {
	registerOnce.Do(func() {
		registerErr = mysql.Register(runtime.EnvironmentTypeCompose)
	})
	if registerErr != nil {
		return nil, registerErr
	}

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
