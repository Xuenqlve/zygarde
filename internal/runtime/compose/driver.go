package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuenqlve/zygarde/internal/deployment"
	deploycompose "github.com/xuenqlve/zygarde/internal/deployment/compose"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/render"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

const defaultWorkspaceRoot = ".zygarde/environments"

// Driver implements the runtime.Driver contract for Docker Compose.
type Driver struct {
	workspaceRoot string
	renderer      render.Renderer
	executor      deployment.Executor
}

// NewDriver creates a compose runtime driver.
func NewDriver(workspaceRoot string, renderer render.Renderer, executor deployment.Executor) Driver {
	if workspaceRoot == "" {
		workspaceRoot = defaultWorkspaceRoot
	}
	if renderer == nil {
		renderer = render.NewComposeRenderer()
	}
	if executor == nil {
		executor = deploycompose.NewExecutor(nil)
	}
	return Driver{
		workspaceRoot: workspaceRoot,
		renderer:      renderer,
		executor:      executor,
	}
}

// Type returns the compose runtime type.
func (Driver) Type() runtime.EnvironmentType {
	return runtime.EnvironmentTypeCompose
}

// Prepare initializes the compose runtime layout and environment metadata.
func (d Driver) Prepare(_ context.Context, req runtime.PrepareRequest) (*runtime.PreparedRuntime, error) {
	name := req.Blueprint.Name
	if name == "" {
		name = "zygarde"
	}

	environmentID := buildEnvironmentID(name)
	rootDir := filepath.Join(d.workspaceRoot, environmentID)
	layout := model.RuntimeLayout{
		RootDir:      rootDir,
		RenderDir:    rootDir,
		ComposeFile:  filepath.Join(rootDir, "docker-compose.yaml"),
		MetadataFile: filepath.Join(rootDir, "environment.json"),
		LogsDir:      filepath.Join(rootDir, "logs"),
		DataDir:      filepath.Join(rootDir, "data"),
	}

	for _, dir := range []string{layout.RootDir, layout.RenderDir, layout.LogsDir, layout.DataDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	projectName := req.Blueprint.Runtime.ProjectName
	if projectName == "" {
		projectName = "zygarde-" + environmentID
	}

	return &runtime.PreparedRuntime{
		Environment: model.Environment{
			ID:               environmentID,
			Name:             name,
			BlueprintName:    req.Blueprint.Name,
			BlueprintVersion: req.Blueprint.Version,
			RuntimeType:      string(runtime.EnvironmentTypeCompose),
			Status:           model.EnvironmentStatusCreating,
			ProjectName:      projectName,
			WorkspaceDir:     layout.RootDir,
			ComposeFile:      layout.ComposeFile,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		Layout: layout,
	}, nil
}

// Render generates the compose artifact and writes it to disk.
func (d Driver) Render(ctx context.Context, req runtime.RenderRequest) (*model.RenderResult, error) {
	result, err := d.renderer.Render(ctx, render.Request{
		Prepared: req.Prepared,
		Contexts: req.Contexts,
	})
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(req.Prepared.Layout.ComposeFile, []byte(result.Content), 0o644); err != nil {
		return nil, err
	}
	result.PrimaryFile = req.Prepared.Layout.ComposeFile
	return result, nil
}

// Apply runs the compose deployment stub.
func (d Driver) Apply(ctx context.Context, req runtime.ApplyRequest) (*runtime.OperationResult, error) {
	return d.executor.Apply(ctx, req.Prepared.Environment, req.Rendered)
}

// Status delegates to the compose executor.
func (d Driver) Status(ctx context.Context, env model.Environment) (*runtime.StatusResult, error) {
	return d.executor.Status(ctx, env)
}

// Start delegates to the compose executor.
func (d Driver) Start(ctx context.Context, env model.Environment) (*runtime.OperationResult, error) {
	return d.executor.Start(ctx, env)
}

// Stop delegates to the compose executor.
func (d Driver) Stop(ctx context.Context, env model.Environment) (*runtime.OperationResult, error) {
	return d.executor.Stop(ctx, env)
}

// Destroy delegates to the compose executor.
func (d Driver) Destroy(ctx context.Context, env model.Environment) (*runtime.OperationResult, error) {
	return d.executor.Destroy(ctx, env)
}

// Cleanup delegates to the compose executor.
func (d Driver) Cleanup(ctx context.Context, env model.Environment) (*runtime.OperationResult, error) {
	return d.executor.Cleanup(ctx, env)
}

func buildEnvironmentID(name string) string {
	normalized := strings.ToLower(name)
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		normalized = "zygarde"
	}
	return fmt.Sprintf("%s-%d", normalized, time.Now().Unix())
}
