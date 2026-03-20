package lifecycle_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	coordpkg "github.com/xuenqlve/zygarde/internal/coordinator"
	"github.com/xuenqlve/zygarde/internal/environment"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	"github.com/xuenqlve/zygarde/internal/store"
)

func TestStartRejectsDestroyedEnvironment(t *testing.T) {
	coord, _, driver := newLifecycleCoordinator(t, model.EnvironmentStatusDestroyed)

	_, err := coord.Start(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-test"})
	if err == nil {
		t.Fatal("expected start error")
	}
	if !strings.Contains(err.Error(), "cannot be started from status destroyed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if driver.startCalls != 0 {
		t.Fatalf("expected start not to reach driver, got %d calls", driver.startCalls)
	}
}

func TestStopRejectsDestroyedEnvironment(t *testing.T) {
	coord, _, driver := newLifecycleCoordinator(t, model.EnvironmentStatusDestroyed)

	_, err := coord.Stop(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-test"})
	if err == nil {
		t.Fatal("expected stop error")
	}
	if !strings.Contains(err.Error(), "cannot be stopped from status destroyed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if driver.stopCalls != 0 {
		t.Fatalf("expected stop not to reach driver, got %d calls", driver.stopCalls)
	}
}

func TestDoctorRejectsDestroyedEnvironment(t *testing.T) {
	coord, _, driver := newLifecycleCoordinator(t, model.EnvironmentStatusDestroyed)

	_, err := coord.Doctor(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-test"})
	if err == nil {
		t.Fatal("expected doctor error")
	}
	if !strings.Contains(err.Error(), "cannot be diagnosed from status destroyed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if driver.doctorCalls != 0 {
		t.Fatalf("expected doctor not to reach driver, got %d calls", driver.doctorCalls)
	}
}

func TestDownRejectsDestroyedEnvironment(t *testing.T) {
	coord, _, driver := newLifecycleCoordinator(t, model.EnvironmentStatusDestroyed)

	_, err := coord.Down(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-test"})
	if err == nil {
		t.Fatal("expected down error")
	}
	if !strings.Contains(err.Error(), "cannot be taken down from status destroyed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if driver.destroyCalls != 0 || driver.cleanupCalls != 0 {
		t.Fatalf("expected down not to reach driver, got destroy=%d cleanup=%d", driver.destroyCalls, driver.cleanupCalls)
	}
}

func TestStatusReturnsErrorWhenRuntimeArtifactMissing(t *testing.T) {
	coord, envStore, _ := newLifecycleCoordinator(t, model.EnvironmentStatusRunning)
	delete(envStore.artifacts, "env-test")

	_, err := coord.Status(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-test"})
	if err == nil {
		t.Fatal("expected missing artifact error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestStatusDriverFailureDoesNotOverwritePersistedStatus(t *testing.T) {
	coord, envStore, driver := newLifecycleCoordinator(t, model.EnvironmentStatusRunning)
	driver.statusErr = errors.New("status boom")

	_, err := coord.Status(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-test"})
	if err == nil {
		t.Fatal("expected status error")
	}
	if !strings.Contains(err.Error(), "status boom") {
		t.Fatalf("unexpected error: %v", err)
	}

	env, getErr := envStore.Get("env-test")
	if getErr != nil {
		t.Fatalf("reload environment: %v", getErr)
	}
	if env.Status != model.EnvironmentStatusRunning {
		t.Fatalf("expected persisted status to remain running, got %s", env.Status)
	}
}

func newLifecycleCoordinator(t *testing.T, status model.EnvironmentStatus) (coordpkg.Coordinator, *memoryEnvironmentStore, *fakeLifecycleDriver) {
	t.Helper()

	driver := &fakeLifecycleDriver{}
	registry, err := runtime.NewRegistry(driver)
	if err != nil {
		t.Fatalf("new runtime registry: %v", err)
	}

	envStore := newMemoryEnvironmentStore()
	env := model.Environment{
		ID:            "env-test",
		Name:          "env-test",
		BlueprintName: "bp-test",
		RuntimeType:   string(runtime.EnvironmentTypeCompose),
		Status:        status,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := envStore.Save(env); err != nil {
		t.Fatalf("save environment fixture: %v", err)
	}
	if err := envStore.SaveRuntimeArtifact(runtime.RuntimeArtifact{
		EnvironmentID: env.ID,
		RuntimeType:   runtime.EnvironmentTypeCompose,
		WorkspaceDir:  "/tmp/" + env.ID,
		ProjectName:   "project-" + env.ID,
		PrimaryFile:   "/tmp/" + env.ID + "/docker-compose.yml",
	}); err != nil {
		t.Fatalf("save artifact fixture: %v", err)
	}

	return coordpkg.New(fakeBlueprintStore{}, envStore, registry), envStore, driver
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
	statusErr    error
	doctorCalls  int
	startCalls   int
	stopCalls    int
	destroyCalls int
	cleanupCalls int
}

func (d *fakeLifecycleDriver) Type() runtime.EnvironmentType { return runtime.EnvironmentTypeCompose }
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
	if d.statusErr != nil {
		return nil, d.statusErr
	}
	return &runtime.StatusResult{Status: model.EnvironmentStatusRunning, Message: "ok"}, nil
}
func (d *fakeLifecycleDriver) Doctor(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.doctorCalls++
	return &runtime.OperationResult{Message: "doctor ok"}, nil
}
func (d *fakeLifecycleDriver) Start(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.startCalls++
	return &runtime.OperationResult{Message: "start ok"}, nil
}
func (d *fakeLifecycleDriver) Stop(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.stopCalls++
	return &runtime.OperationResult{Message: "stop ok"}, nil
}
func (d *fakeLifecycleDriver) Destroy(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.destroyCalls++
	return &runtime.OperationResult{Message: "destroy ok"}, nil
}
func (d *fakeLifecycleDriver) Cleanup(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.cleanupCalls++
	return &runtime.OperationResult{Message: "cleanup ok"}, nil
}

var _ store.BlueprintStore = fakeBlueprintStore{}
var _ environment.Store = (*memoryEnvironmentStore)(nil)
var _ runtime.Driver = (*fakeLifecycleDriver)(nil)
