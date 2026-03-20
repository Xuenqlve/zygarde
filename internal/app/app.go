package app

import (
	"context"
	"fmt"

	_ "github.com/xuenqlve/zygarde/pkg/register"

	"github.com/xuenqlve/zygarde/internal/config"
	"github.com/xuenqlve/zygarde/internal/coordinator"
	deploycompose "github.com/xuenqlve/zygarde/internal/deployment/compose"
	"github.com/xuenqlve/zygarde/internal/environment"
	"github.com/xuenqlve/zygarde/internal/render"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	"github.com/xuenqlve/zygarde/internal/store"
)

// App wires the create flow dependencies together.
type App struct {
	cfg         config.Config
	coordinator coordinator.Coordinator
}

// New creates an application instance with the default local dependencies.
func New() (*App, error) {
	cfg := config.Default()
	runtimes, err := runtime.NewRegistry(
		runtimecompose.NewDriver(
			"",
			render.NewComposeRenderer(cfg.ContainerEngine),
			deploycompose.NewExecutor(cfg.ContainerEngine, nil),
		),
	)
	if err != nil {
		return nil, err
	}

	return NewWithCoordinator(
		cfg,
		coordinator.New(
			store.NewFileBlueprintStore(),
			environment.NewFileStore(".zygarde/environments"),
			runtimes,
		),
	), nil
}

// NewWithCoordinator creates an application instance with injected dependencies.
func NewWithCoordinator(cfg config.Config, coord coordinator.Coordinator) *App {
	return &App{
		cfg:         cfg,
		coordinator: coord,
	}
}

// Create is reserved for the future create-only flow.
func (a *App) Create(ctx context.Context, blueprintFile string, envType runtime.EnvironmentType) (*coordinator.CreateResult, error) {
	if envType == "" {
		envType = a.cfg.DefaultEnvironmentType
	}

	result, err := a.coordinator.Create(ctx, coordinator.CreateRequest{
		BlueprintFile:   blueprintFile,
		EnvironmentType: envType,
	})
	if err != nil {
		return nil, err
	}
	if err := environment.SaveCurrent(environment.CurrentEnvironment{
		EnvironmentID: result.EnvironmentID,
		WorkspaceDir:  result.WorkspaceDir,
		ProjectName:   result.ProjectName,
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// Up creates runtime assets and starts the environment.
func (a *App) Up(ctx context.Context, blueprintFile string, envType runtime.EnvironmentType) (*coordinator.CreateResult, error) {
	if envType == "" {
		envType = a.cfg.DefaultEnvironmentType
	}

	result, err := a.coordinator.Up(ctx, coordinator.UpRequest{
		BlueprintFile:   blueprintFile,
		EnvironmentType: envType,
	})
	if err != nil {
		return nil, err
	}
	if err := environment.SaveCurrent(environment.CurrentEnvironment{
		EnvironmentID: result.EnvironmentID,
		WorkspaceDir:  result.WorkspaceDir,
		ProjectName:   result.ProjectName,
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// List returns all persisted environments.
func (a *App) List(ctx context.Context) (*coordinator.ListResult, error) {
	return a.coordinator.List(ctx)
}

// ListBlueprints returns discovered local blueprint files.
func (a *App) ListBlueprints(ctx context.Context, root string) (*coordinator.BlueprintListResult, error) {
	return a.coordinator.ListBlueprints(ctx, root)
}

// ListTemplates returns the built-in middleware template catalog.
func (a *App) ListTemplates(ctx context.Context, envType runtime.EnvironmentType) (*coordinator.TemplateListResult, error) {
	if envType == "" {
		envType = a.cfg.DefaultEnvironmentType
	}
	return a.coordinator.ListTemplates(ctx, string(envType))
}

// ShowTemplate returns one built-in middleware template detail.
func (a *App) ShowTemplate(ctx context.Context, middleware, templateName string, envType runtime.EnvironmentType) (*coordinator.TemplateShowResult, error) {
	if envType == "" {
		envType = a.cfg.DefaultEnvironmentType
	}
	return a.coordinator.ShowTemplate(ctx, middleware, templateName, string(envType))
}

// ShowBlueprint returns one blueprint summary.
func (a *App) ShowBlueprint(ctx context.Context, blueprintFile string, envType runtime.EnvironmentType) (*coordinator.BlueprintShowResult, error) {
	if envType == "" {
		envType = a.cfg.DefaultEnvironmentType
	}
	return a.coordinator.ShowBlueprint(ctx, blueprintFile, envType)
}

// ValidateBlueprint validates one blueprint for the target runtime.
func (a *App) ValidateBlueprint(ctx context.Context, blueprintFile string, envType runtime.EnvironmentType) (*coordinator.BlueprintValidateResult, error) {
	if envType == "" {
		envType = a.cfg.DefaultEnvironmentType
	}
	return a.coordinator.ValidateBlueprint(ctx, blueprintFile, envType)
}

// Status queries one created environment by id.
func (a *App) Status(ctx context.Context, environmentID string) (*coordinator.Result, error) {
	environmentID, err := a.resolveEnvironmentID(environmentID)
	if err != nil {
		return nil, err
	}
	return a.coordinator.Status(ctx, coordinator.EnvironmentRequest{EnvironmentID: environmentID})
}

// Doctor checks one created environment by id.
func (a *App) Doctor(ctx context.Context, environmentID string) (*coordinator.Result, error) {
	environmentID, err := a.resolveEnvironmentID(environmentID)
	if err != nil {
		return nil, err
	}
	return a.coordinator.Doctor(ctx, coordinator.EnvironmentRequest{EnvironmentID: environmentID})
}

// Start starts one created environment by id.
func (a *App) Start(ctx context.Context, environmentID string) (*coordinator.Result, error) {
	environmentID, err := a.resolveEnvironmentID(environmentID)
	if err != nil {
		return nil, err
	}
	return a.coordinator.Start(ctx, coordinator.EnvironmentRequest{EnvironmentID: environmentID})
}

// Stop stops one created environment by id.
func (a *App) Stop(ctx context.Context, environmentID string) (*coordinator.Result, error) {
	environmentID, err := a.resolveEnvironmentID(environmentID)
	if err != nil {
		return nil, err
	}
	return a.coordinator.Stop(ctx, coordinator.EnvironmentRequest{EnvironmentID: environmentID})
}

// Down stops and removes one created environment by id.
func (a *App) Down(ctx context.Context, environmentID string) (*coordinator.Result, error) {
	environmentID, err := a.resolveEnvironmentID(environmentID)
	if err != nil {
		return nil, err
	}
	result, err := a.coordinator.Down(ctx, coordinator.EnvironmentRequest{EnvironmentID: environmentID})
	if err != nil {
		return nil, err
	}
	current, currentErr := environment.LoadCurrent()
	if currentErr == nil && current.EnvironmentID == environmentID {
		if err := environment.ClearCurrent(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// Destroy destroys one created environment by id.
func (a *App) Destroy(ctx context.Context, environmentID string) (*coordinator.Result, error) {
	return a.Down(ctx, environmentID)
}

func (a *App) resolveEnvironmentID(environmentID string) (string, error) {
	if environmentID != "" {
		return environmentID, nil
	}
	current, err := environment.LoadCurrent()
	if err != nil {
		return "", fmt.Errorf("resolve current environment: %w", err)
	}
	return current.EnvironmentID, nil
}
