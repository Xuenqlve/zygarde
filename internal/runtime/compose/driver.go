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
		renderer = render.NewComposeRenderer("")
	}
	if executor == nil {
		executor = deploycompose.NewExecutor("", nil)
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
func (d Driver) Prepare(_ context.Context, req runtime.PrepareRequest) (*runtime.PreparePlan, error) {
	name := req.Input.RequestedName
	if name == "" {
		name = req.Input.BlueprintName
	}
	if name == "" {
		name = "zygarde"
	}

	environmentID := buildEnvironmentID(name)
	workspaceRoot := req.Input.WorkspaceRoot
	if workspaceRoot == "" {
		workspaceRoot = d.workspaceRoot
	}
	rootDir := filepath.Join(workspaceRoot, environmentID)
	layout := model.RuntimeLayout{
		RootDir:      rootDir,
		RenderDir:    rootDir,
		ComposeFile:  filepath.Join(rootDir, "docker-compose.yml"),
		EnvFile:      filepath.Join(rootDir, ".env"),
		BuildScript:  filepath.Join(rootDir, "build.sh"),
		CheckScript:  filepath.Join(rootDir, "check.sh"),
		ReadmeFile:   filepath.Join(rootDir, "README.md"),
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
	projectName := req.Input.ProjectName
	if projectName == "" {
		projectName = "zygarde-" + environmentID
	}

	return &runtime.PreparePlan{
		Environment: model.Environment{
			ID:               environmentID,
			Name:             name,
			BlueprintName:    req.Input.BlueprintName,
			BlueprintVersion: req.Input.BlueprintVersion,
			RuntimeType:      string(req.Input.RuntimeType),
			Status:           model.EnvironmentStatusCreating,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		Layout: layout,
		Files: map[string]string{
			"compose_file": layout.ComposeFile,
			"env_file":     layout.EnvFile,
			"build_script": layout.BuildScript,
			"check_script": layout.CheckScript,
			"readme_file":  layout.ReadmeFile,
		},
		ProjectName: projectName,
	}, nil
}

// Render generates the compose artifact and writes it to disk.
func (d Driver) Render(ctx context.Context, req runtime.RenderRequest) (*runtime.RenderPlan, error) {
	return d.renderer.Render(ctx, render.Request{
		Prepared: req.Prepared,
		Contexts: req.Contexts,
	})
}

// PlanApply aggregates one compose apply plan from rendered assets and runtime contexts.
func (d Driver) PlanApply(_ context.Context, req runtime.BuildApplyRequest) (*runtime.ApplyPlan, error) {
	serviceNames := make([]string, 0, len(req.Contexts))
	for _, contextItem := range req.Contexts {
		item, ok := contextItem.(runtime.ComposeContext)
		if !ok {
			return nil, fmt.Errorf("compose apply: unsupported context type %T", contextItem)
		}
		applyInput := item.ApplyInput()
		if applyInput.ServiceName == "" {
			return nil, fmt.Errorf("compose apply: service name is required")
		}
		serviceNames = append(serviceNames, applyInput.ServiceName)
	}
	if len(serviceNames) == 0 {
		return nil, fmt.Errorf("compose apply: at least one runtime context is required")
	}

	return &runtime.ApplyPlan{
		Environment:  req.Prepared.Environment,
		WorkspaceDir: req.Prepared.Layout.RootDir,
		ProjectName:  req.Prepared.ProjectName,
		PrimaryFile:  req.Rendered.PrimaryFile,
		BuildScript:  req.Rendered.BuildScript,
		CheckScript:  req.Rendered.CheckScript,
		Services:     serviceNames,
	}, nil
}

// Apply runs the compose deployment plan.
func (d Driver) Apply(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error) {
	return d.executor.Apply(ctx, plan)
}

// Create runs the compose create plan without starting containers.
func (d Driver) Create(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error) {
	return d.executor.Create(ctx, plan)
}

// PlanLifecycle aggregates one compose lifecycle plan from persisted environment metadata.
func (d Driver) PlanLifecycle(_ context.Context, req runtime.BuildLifecycleRequest) (*runtime.LifecyclePlan, error) {
	env := req.Environment
	return &runtime.LifecyclePlan{
		Environment:  env,
		WorkspaceDir: req.Artifact.WorkspaceDir,
		ProjectName:  req.Artifact.ProjectName,
		PrimaryFile:  req.Artifact.PrimaryFile,
	}, nil
}

// Status delegates to the compose executor.
func (d Driver) Status(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.StatusResult, error) {
	return d.executor.Status(ctx, plan)
}

// Doctor delegates to the compose executor.
func (d Driver) Doctor(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return d.executor.Doctor(ctx, plan)
}

// Start delegates to the compose executor.
func (d Driver) Start(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return d.executor.Start(ctx, plan)
}

// Stop delegates to the compose executor.
func (d Driver) Stop(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return d.executor.Stop(ctx, plan)
}

// Destroy delegates to the compose executor.
func (d Driver) Destroy(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return d.executor.Destroy(ctx, plan)
}

// Cleanup delegates to the compose executor.
func (d Driver) Cleanup(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return d.executor.Cleanup(ctx, plan)
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
