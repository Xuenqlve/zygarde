package runtime

import (
	"context"

	"github.com/xuenqlve/zygarde/internal/model"
)

// PrepareRequest contains the normalized input for runtime initialization.
type PrepareRequest struct {
	Input    PrepareInput
	Contexts []EnvironmentContext
}

// PrepareInput contains the environment-level data required by the prepare stage.
type PrepareInput struct {
	BlueprintName    string
	BlueprintVersion string
	RuntimeType      EnvironmentType
	RequestedName    string
	ProjectName      string
	WorkspaceRoot    string
}

// PreparePlan contains the runtime layout and initialized environment metadata.
type PreparePlan struct {
	Environment model.Environment
	Layout      model.RuntimeLayout
	Files       map[string]string
	ProjectName string
}

// RuntimeArtifact describes runtime-specific persisted execution metadata.
type RuntimeArtifact struct {
	EnvironmentID string
	RuntimeType   EnvironmentType
	WorkspaceDir  string
	ProjectName   string
	PrimaryFile   string
	Files         map[string]string
}

// RenderedAsset describes one generated runtime asset file.
type RenderedAsset struct {
	Path string
	Mode int
}

// RenderPlan describes the output of the render stage.
type RenderPlan struct {
	Prepared       PreparePlan
	Content        string
	PrimaryFile    string
	BuildScript    string
	CheckScript    string
	ComposeVersion string
	Services       []string
	Assets         []RenderedAsset
	Warnings       []string
}

// RenderRequest contains the input needed to generate runtime artifacts.
type RenderRequest struct {
	Prepared PreparePlan
	Contexts []EnvironmentContext
}

// BuildApplyRequest contains the data required to aggregate one apply-stage plan.
type BuildApplyRequest struct {
	Prepared PreparePlan
	Rendered RenderPlan
	Contexts []EnvironmentContext
}

// ApplyPlan contains the minimal runtime-specific inputs required by executor.Apply.
type ApplyPlan struct {
	Environment  model.Environment
	WorkspaceDir string
	ProjectName  string
	PrimaryFile  string
	BuildScript  string
	CheckScript  string
	Services     []string
}

// BuildLifecycleRequest contains the persisted environment data required to build one lifecycle plan.
type BuildLifecycleRequest struct {
	Environment model.Environment
	Artifact    RuntimeArtifact
}

// LifecyclePlan contains the minimal persisted runtime data required by status/start/stop/destroy/cleanup.
type LifecyclePlan struct {
	Environment  model.Environment
	WorkspaceDir string
	ProjectName  string
	PrimaryFile  string
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
	Prepare(ctx context.Context, req PrepareRequest) (*PreparePlan, error)
	Render(ctx context.Context, req RenderRequest) (*RenderPlan, error)
	PlanApply(ctx context.Context, req BuildApplyRequest) (*ApplyPlan, error)
	Apply(ctx context.Context, plan ApplyPlan) (*OperationResult, error)
	PlanLifecycle(ctx context.Context, req BuildLifecycleRequest) (*LifecyclePlan, error)
	Status(ctx context.Context, plan LifecyclePlan) (*StatusResult, error)
	Start(ctx context.Context, plan LifecyclePlan) (*OperationResult, error)
	Stop(ctx context.Context, plan LifecyclePlan) (*OperationResult, error)
	Destroy(ctx context.Context, plan LifecyclePlan) (*OperationResult, error)
	Cleanup(ctx context.Context, plan LifecyclePlan) (*OperationResult, error)
}
