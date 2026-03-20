package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	apppkg "github.com/xuenqlve/zygarde/internal/app"
	"github.com/xuenqlve/zygarde/internal/config"
	"github.com/xuenqlve/zygarde/internal/coordinator"
	"github.com/xuenqlve/zygarde/internal/environment"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	"github.com/xuenqlve/zygarde/internal/store"
)

func TestStatusWithoutCurrentEnvironmentReturnsHelpfulError(t *testing.T) {
	application := newTestApp(t)

	_, err := application.Status(context.Background(), "")
	if err == nil {
		t.Fatal("expected error when current environment marker is missing")
	}
	if !strings.Contains(err.Error(), "resolve current environment") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusUsesCurrentEnvironmentMarkerWhenIDMissing(t *testing.T) {
	application, envStore, driver := newTestAppWithRuntime(t)
	writeCurrentEnvironment(t, "env-current")
	saveEnvironmentFixture(t, envStore, "env-current")

	result, err := application.Status(context.Background(), "")
	if err != nil {
		t.Fatalf("status with current environment: %v", err)
	}
	if !strings.Contains(result.Message, "environment env-current status: running") {
		t.Fatalf("unexpected status message: %q", result.Message)
	}
	if driver.statusCalls != 1 {
		t.Fatalf("expected one status call, got %d", driver.statusCalls)
	}
}

func TestDownClearsCurrentEnvironmentMarkerWhenTargetMatches(t *testing.T) {
	application, envStore, driver := newTestAppWithRuntime(t)
	writeCurrentEnvironment(t, "env-current")
	saveEnvironmentFixture(t, envStore, "env-current")

	result, err := application.Down(context.Background(), "")
	if err != nil {
		t.Fatalf("down with current environment: %v", err)
	}
	if !strings.Contains(result.Message, "environment env-current down completed") {
		t.Fatalf("unexpected down message: %q", result.Message)
	}
	if !driver.destroyCalled || !driver.cleanupCalled {
		t.Fatalf("expected destroy and cleanup to be called, got destroy=%t cleanup=%t", driver.destroyCalled, driver.cleanupCalled)
	}
	if _, err := environment.LoadCurrent(); err == nil {
		t.Fatal("expected current environment marker to be removed")
	}
}

func TestDownPreservesCurrentEnvironmentMarkerWhenExplicitIDDiffers(t *testing.T) {
	application, envStore, driver := newTestAppWithRuntime(t)
	writeCurrentEnvironment(t, "env-current")
	saveEnvironmentFixture(t, envStore, "env-current")
	saveEnvironmentFixture(t, envStore, "env-other")

	result, err := application.Down(context.Background(), "env-other")
	if err != nil {
		t.Fatalf("down with explicit environment: %v", err)
	}
	if !strings.Contains(result.Message, "environment env-other down completed") {
		t.Fatalf("unexpected down message: %q", result.Message)
	}
	if !driver.destroyCalled || !driver.cleanupCalled {
		t.Fatalf("expected destroy and cleanup to be called, got destroy=%t cleanup=%t", driver.destroyCalled, driver.cleanupCalled)
	}

	current, err := environment.LoadCurrent()
	if err != nil {
		t.Fatalf("expected current environment marker to remain: %v", err)
	}
	if current.EnvironmentID != "env-current" {
		t.Fatalf("expected current environment marker to remain unchanged, got %q", current.EnvironmentID)
	}
}

func newTestApp(t *testing.T) *apppkg.App {
	t.Helper()
	application, _, _ := newTestAppWithRuntime(t)
	return application
}

func newTestAppWithRuntime(t *testing.T) (*apppkg.App, *memoryEnvironmentStore, *fakeLifecycleDriver) {
	t.Helper()
	restore := chdirForTest(t, t.TempDir())
	t.Cleanup(restore)

	driver := &fakeLifecycleDriver{}
	registry, err := runtime.NewRegistry(driver)
	if err != nil {
		t.Fatalf("new runtime registry: %v", err)
	}

	envStore := newMemoryEnvironmentStore()
	application := apppkg.NewWithCoordinator(
		config.Default(),
		coordinator.New(fakeBlueprintStore{}, envStore, registry),
	)
	return application, envStore, driver
}

func writeCurrentEnvironment(t *testing.T, id string) {
	t.Helper()
	if err := environment.SaveCurrent(environment.CurrentEnvironment{
		EnvironmentID: id,
		WorkspaceDir:  filepath.Join(t.TempDir(), id),
		ProjectName:   "project-" + id,
	}); err != nil {
		t.Fatalf("save current environment: %v", err)
	}
}

func saveEnvironmentFixture(t *testing.T, envStore *memoryEnvironmentStore, id string) {
	t.Helper()
	env := model.Environment{
		ID:            id,
		Name:          id,
		BlueprintName: id,
		RuntimeType:   string(runtime.EnvironmentTypeCompose),
		Status:        model.EnvironmentStatusRunning,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := envStore.Save(env); err != nil {
		t.Fatalf("save environment fixture: %v", err)
	}
	if err := envStore.SaveRuntimeArtifact(runtime.RuntimeArtifact{
		EnvironmentID: id,
		RuntimeType:   runtime.EnvironmentTypeCompose,
		WorkspaceDir:  filepath.Join(t.TempDir(), id),
		ProjectName:   "project-" + id,
		PrimaryFile:   filepath.Join(t.TempDir(), id, "docker-compose.yml"),
	}); err != nil {
		t.Fatalf("save runtime artifact fixture: %v", err)
	}
}

func chdirForTest(t *testing.T, dir string) func() {
	t.Helper()

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	return func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}
}

type fakeBlueprintStore struct{}

func (fakeBlueprintStore) LoadBlueprint(string) (model.Blueprint, error) {
	return model.Blueprint{}, nil
}

func (fakeBlueprintStore) ListBlueprints(string) ([]store.BlueprintFile, error) {
	return []store.BlueprintFile{}, nil
}

type memoryEnvironmentStore struct {
	environments map[string]model.Environment
	artifacts    map[string]runtime.RuntimeArtifact
}

func newMemoryEnvironmentStore() *memoryEnvironmentStore {
	return &memoryEnvironmentStore{
		environments: make(map[string]model.Environment),
		artifacts:    make(map[string]runtime.RuntimeArtifact),
	}
}

func (s *memoryEnvironmentStore) Save(env model.Environment) error {
	s.environments[env.ID] = env
	return nil
}

func (s *memoryEnvironmentStore) Get(id string) (model.Environment, error) {
	env, ok := s.environments[id]
	if !ok {
		return model.Environment{}, os.ErrNotExist
	}
	return env, nil
}

func (s *memoryEnvironmentStore) List() ([]model.Environment, error) {
	items := make([]model.Environment, 0, len(s.environments))
	for _, env := range s.environments {
		items = append(items, env)
	}
	return items, nil
}

func (s *memoryEnvironmentStore) SaveRuntimeArtifact(artifact runtime.RuntimeArtifact) error {
	s.artifacts[artifact.EnvironmentID] = artifact
	return nil
}

func (s *memoryEnvironmentStore) GetRuntimeArtifact(id string) (runtime.RuntimeArtifact, error) {
	artifact, ok := s.artifacts[id]
	if !ok {
		return runtime.RuntimeArtifact{}, os.ErrNotExist
	}
	return artifact, nil
}

type fakeLifecycleDriver struct {
	statusCalls   int
	destroyCalled bool
	cleanupCalled bool
}

func (d *fakeLifecycleDriver) Type() runtime.EnvironmentType {
	return runtime.EnvironmentTypeCompose
}

func (d *fakeLifecycleDriver) Prepare(context.Context, runtime.PrepareRequest) (*runtime.PreparePlan, error) {
	return nil, nil
}

func (d *fakeLifecycleDriver) Render(context.Context, runtime.RenderRequest) (*runtime.RenderPlan, error) {
	return nil, nil
}

func (d *fakeLifecycleDriver) PlanApply(context.Context, runtime.BuildApplyRequest) (*runtime.ApplyPlan, error) {
	return nil, nil
}

func (d *fakeLifecycleDriver) Apply(context.Context, runtime.ApplyPlan) (*runtime.OperationResult, error) {
	return nil, nil
}

func (d *fakeLifecycleDriver) Create(context.Context, runtime.ApplyPlan) (*runtime.OperationResult, error) {
	return &runtime.OperationResult{Message: "create ok", Changed: true}, nil
}

func (d *fakeLifecycleDriver) PlanLifecycle(_ context.Context, req runtime.BuildLifecycleRequest) (*runtime.LifecyclePlan, error) {
	return &runtime.LifecyclePlan{
		Environment:  req.Environment,
		WorkspaceDir: req.Artifact.WorkspaceDir,
		ProjectName:  req.Artifact.ProjectName,
		PrimaryFile:  req.Artifact.PrimaryFile,
	}, nil
}

func (d *fakeLifecycleDriver) Status(context.Context, runtime.LifecyclePlan) (*runtime.StatusResult, error) {
	d.statusCalls++
	return &runtime.StatusResult{
		Status:  model.EnvironmentStatusRunning,
		Message: "status ok",
	}, nil
}

func (d *fakeLifecycleDriver) Doctor(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return &runtime.OperationResult{Message: "doctor ok"}, nil
}

func (d *fakeLifecycleDriver) Start(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return &runtime.OperationResult{Message: "start ok"}, nil
}

func (d *fakeLifecycleDriver) Stop(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return &runtime.OperationResult{Message: "stop ok"}, nil
}

func (d *fakeLifecycleDriver) Destroy(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.destroyCalled = true
	return &runtime.OperationResult{Message: "destroy ok", Changed: true}, nil
}

func (d *fakeLifecycleDriver) Cleanup(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.cleanupCalled = true
	return &runtime.OperationResult{Message: "cleanup ok", Changed: true}, nil
}

var _ store.BlueprintStore = fakeBlueprintStore{}
var _ environment.Store = (*memoryEnvironmentStore)(nil)
var _ runtime.Driver = (*fakeLifecycleDriver)(nil)
