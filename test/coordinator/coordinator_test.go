package coordinator_test

import (
	"context"
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

func TestListReturnsPersistedEnvironments(t *testing.T) {
	coord, _, _ := newTestCoordinator(t)

	result, err := coord.List(context.Background())
	if err != nil {
		t.Fatalf("list environments: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 environments, got %d", len(result.Items))
	}
	if result.Items[0].ID != "env-running" {
		t.Fatalf("expected env-running first, got %q", result.Items[0].ID)
	}
	if result.Items[1].ID != "env-stopped" {
		t.Fatalf("expected env-stopped second, got %q", result.Items[1].ID)
	}
}

func TestStatusUpdatesEnvironmentStatusAndEndpoints(t *testing.T) {
	coord, envStore, driver := newTestCoordinator(t)

	result, err := coord.Status(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-running"})
	if err != nil {
		t.Fatalf("status environment: %v", err)
	}
	if !strings.Contains(result.Message, "environment env-running status: running") {
		t.Fatalf("unexpected status message: %q", result.Message)
	}
	if driver.statusCalls != 1 {
		t.Fatalf("expected one status call, got %d", driver.statusCalls)
	}

	env, err := envStore.Get("env-running")
	if err != nil {
		t.Fatalf("reload environment: %v", err)
	}
	if env.Status != model.EnvironmentStatusRunning {
		t.Fatalf("expected running status, got %s", env.Status)
	}
	if len(env.Endpoints) != 1 || env.Endpoints[0].Port != 3306 {
		t.Fatalf("expected persisted endpoint update, got %+v", env.Endpoints)
	}
}

func TestDoctorReturnsUserFacingMessage(t *testing.T) {
	coord, _, driver := newTestCoordinator(t)

	result, err := coord.Doctor(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-running"})
	if err != nil {
		t.Fatalf("doctor environment: %v", err)
	}
	if !strings.Contains(result.Message, "environment env-running doctor passed: doctor ok") {
		t.Fatalf("unexpected doctor message: %q", result.Message)
	}
	if driver.doctorCalls != 1 {
		t.Fatalf("expected one doctor call, got %d", driver.doctorCalls)
	}
}

func TestStartMarksEnvironmentRunning(t *testing.T) {
	coord, envStore, driver := newTestCoordinator(t)

	result, err := coord.Start(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-stopped"})
	if err != nil {
		t.Fatalf("start environment: %v", err)
	}
	if !strings.Contains(result.Message, "environment env-stopped started: start ok") {
		t.Fatalf("unexpected start message: %q", result.Message)
	}
	if driver.startCalls != 1 {
		t.Fatalf("expected one start call, got %d", driver.startCalls)
	}

	env, err := envStore.Get("env-stopped")
	if err != nil {
		t.Fatalf("reload environment: %v", err)
	}
	if env.Status != model.EnvironmentStatusRunning {
		t.Fatalf("expected running status after start, got %s", env.Status)
	}
	if len(env.Endpoints) != 1 || env.Endpoints[0].Port != 3307 {
		t.Fatalf("expected start endpoints to persist, got %+v", env.Endpoints)
	}
}

func TestStopMarksEnvironmentStopped(t *testing.T) {
	coord, envStore, driver := newTestCoordinator(t)

	result, err := coord.Stop(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-running"})
	if err != nil {
		t.Fatalf("stop environment: %v", err)
	}
	if !strings.Contains(result.Message, "environment env-running stopped: stop ok") {
		t.Fatalf("unexpected stop message: %q", result.Message)
	}
	if driver.stopCalls != 1 {
		t.Fatalf("expected one stop call, got %d", driver.stopCalls)
	}

	env, err := envStore.Get("env-running")
	if err != nil {
		t.Fatalf("reload environment: %v", err)
	}
	if env.Status != model.EnvironmentStatusStopped {
		t.Fatalf("expected stopped status after stop, got %s", env.Status)
	}
}

func TestDownDestroysCleansAndPersistsDestroyedStatus(t *testing.T) {
	coord, envStore, driver := newTestCoordinator(t)

	result, err := coord.Down(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-running"})
	if err != nil {
		t.Fatalf("down environment: %v", err)
	}
	if !strings.Contains(result.Message, "environment env-running down completed: destroy ok; cleanup: cleanup ok") {
		t.Fatalf("unexpected down message: %q", result.Message)
	}
	if !driver.destroyCalled || !driver.cleanupCalled {
		t.Fatalf("expected destroy and cleanup to be called, got destroy=%t cleanup=%t", driver.destroyCalled, driver.cleanupCalled)
	}
	if driver.callOrder != "destroy,cleanup" {
		t.Fatalf("expected destroy then cleanup order, got %q", driver.callOrder)
	}

	env, err := envStore.Get("env-running")
	if err != nil {
		t.Fatalf("reload environment: %v", err)
	}
	if env.Status != model.EnvironmentStatusDestroyed {
		t.Fatalf("expected destroyed status after down, got %s", env.Status)
	}
	if len(env.Endpoints) != 0 {
		t.Fatalf("expected endpoints cleared after down, got %+v", env.Endpoints)
	}
}

func TestDestroyIsAliasOfDown(t *testing.T) {
	coord, envStore, driver := newTestCoordinator(t)

	result, err := coord.Destroy(context.Background(), coordpkg.EnvironmentRequest{EnvironmentID: "env-running"})
	if err != nil {
		t.Fatalf("destroy environment: %v", err)
	}
	if !strings.Contains(result.Message, "environment env-running down completed") {
		t.Fatalf("unexpected destroy message: %q", result.Message)
	}
	if !driver.destroyCalled || !driver.cleanupCalled {
		t.Fatalf("expected destroy and cleanup to be called, got destroy=%t cleanup=%t", driver.destroyCalled, driver.cleanupCalled)
	}

	env, err := envStore.Get("env-running")
	if err != nil {
		t.Fatalf("reload environment: %v", err)
	}
	if env.Status != model.EnvironmentStatusDestroyed {
		t.Fatalf("expected destroyed status after destroy alias, got %s", env.Status)
	}
}

func newTestCoordinator(t *testing.T) (coordpkg.Coordinator, *memoryEnvironmentStore, *fakeLifecycleDriver) {
	t.Helper()

	driver := &fakeLifecycleDriver{}
	registry, err := runtime.NewRegistry(driver)
	if err != nil {
		t.Fatalf("new runtime registry: %v", err)
	}

	envStore := newMemoryEnvironmentStore()
	saveFixture(t, envStore, model.Environment{
		ID:            "env-running",
		Name:          "env-running",
		BlueprintName: "bp-running",
		RuntimeType:   string(runtime.EnvironmentTypeCompose),
		Status:        model.EnvironmentStatusRunning,
		CreatedAt:     time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 3, 18, 10, 5, 0, 0, time.UTC),
	})
	saveFixture(t, envStore, model.Environment{
		ID:            "env-stopped",
		Name:          "env-stopped",
		BlueprintName: "bp-stopped",
		RuntimeType:   string(runtime.EnvironmentTypeCompose),
		Status:        model.EnvironmentStatusStopped,
		CreatedAt:     time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 3, 18, 9, 5, 0, 0, time.UTC),
	})

	return coordpkg.New(fakeBlueprintStore{}, envStore, registry), envStore, driver
}

func saveFixture(t *testing.T, envStore *memoryEnvironmentStore, env model.Environment) {
	t.Helper()
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
	doctorCalls   int
	startCalls    int
	stopCalls     int
	destroyCalled bool
	cleanupCalled bool
	callOrder     string
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
		Endpoints: []model.Endpoint{
			{Name: "mysql", Host: "127.0.0.1", Port: 3306, Protocol: "tcp"},
		},
	}, nil
}

func (d *fakeLifecycleDriver) Doctor(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.doctorCalls++
	return &runtime.OperationResult{Message: "doctor ok"}, nil
}

func (d *fakeLifecycleDriver) Start(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.startCalls++
	return &runtime.OperationResult{
		Message: "start ok",
		Endpoints: []model.Endpoint{
			{Name: "mysql", Host: "127.0.0.1", Port: 3307, Protocol: "tcp"},
		},
	}, nil
}

func (d *fakeLifecycleDriver) Stop(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.stopCalls++
	return &runtime.OperationResult{Message: "stop ok"}, nil
}

func (d *fakeLifecycleDriver) Destroy(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.destroyCalled = true
	d.appendOrder("destroy")
	return &runtime.OperationResult{Message: "destroy ok", Changed: true}, nil
}

func (d *fakeLifecycleDriver) Cleanup(context.Context, runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.cleanupCalled = true
	d.appendOrder("cleanup")
	return &runtime.OperationResult{Message: "cleanup ok", Changed: true}, nil
}

func (d *fakeLifecycleDriver) appendOrder(step string) {
	if d.callOrder == "" {
		d.callOrder = step
		return
	}
	d.callOrder += "," + step
}

var _ store.BlueprintStore = fakeBlueprintStore{}
var _ environment.Store = (*memoryEnvironmentStore)(nil)
var _ runtime.Driver = (*fakeLifecycleDriver)(nil)
