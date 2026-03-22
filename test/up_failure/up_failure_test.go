package up_failure_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuenqlve/zygarde/internal/coordinator"
	"github.com/xuenqlve/zygarde/internal/environment"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	"github.com/xuenqlve/zygarde/internal/store"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

func TestUpRenderFailurePersistsErrorAndCleansWorkspace(t *testing.T) {
	middlewareName := registerTestMiddleware(t)
	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	driver := &fakeDriver{
		workspaceRoot: workspaceRoot,
		renderErr:     errors.New("render boom"),
	}
	registry, err := runtime.NewRegistry(driver)
	if err != nil {
		t.Fatalf("new runtime registry: %v", err)
	}

	envStore := newMemoryEnvironmentStore()
	coord := coordinator.New(
		fakeBlueprintStore{blueprint: testBlueprint(middlewareName)},
		envStore,
		registry,
	)

	_, err = coord.Up(context.Background(), coordinator.UpRequest{
		BlueprintFile:   "ignored.yaml",
		EnvironmentType: runtime.EnvironmentTypeCompose,
	})
	if err == nil {
		t.Fatal("expected up failure")
	}
	if !strings.Contains(err.Error(), "render boom") {
		t.Fatalf("expected render error, got %v", err)
	}

	env, getErr := envStore.Get(driver.environmentID)
	if getErr != nil {
		t.Fatalf("get persisted environment: %v", getErr)
	}
	if env.Status != model.EnvironmentStatusError {
		t.Fatalf("expected error status, got %s", env.Status)
	}
	if !strings.Contains(env.LastError, "render boom") {
		t.Fatalf("expected LastError to contain render failure, got %q", env.LastError)
	}

	if _, getErr := envStore.GetRuntimeArtifact(driver.environmentID); getErr != nil {
		t.Fatalf("expected runtime artifact to be persisted: %v", getErr)
	}
	if !driver.cleanupCalled {
		t.Fatal("expected cleanup to be called")
	}
	if driver.destroyCalled {
		t.Fatal("did not expect destroy to be called for render failure")
	}
}

func TestUpApplyFailureDestroysAndCleansWithErrorState(t *testing.T) {
	middlewareName := registerTestMiddleware(t)
	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	driver := &fakeDriver{
		workspaceRoot: workspaceRoot,
		applyErr:      errors.New("apply boom"),
	}
	registry, err := runtime.NewRegistry(driver)
	if err != nil {
		t.Fatalf("new runtime registry: %v", err)
	}

	envStore := newMemoryEnvironmentStore()
	coord := coordinator.New(
		fakeBlueprintStore{blueprint: testBlueprint(middlewareName)},
		envStore,
		registry,
	)

	_, err = coord.Up(context.Background(), coordinator.UpRequest{
		BlueprintFile:   "ignored.yaml",
		EnvironmentType: runtime.EnvironmentTypeCompose,
	})
	if err == nil {
		t.Fatal("expected up failure")
	}
	if !strings.Contains(err.Error(), "apply boom") {
		t.Fatalf("expected apply error, got %v", err)
	}

	env, getErr := envStore.Get(driver.environmentID)
	if getErr != nil {
		t.Fatalf("get persisted environment: %v", getErr)
	}
	if env.Status != model.EnvironmentStatusError {
		t.Fatalf("expected error status, got %s", env.Status)
	}
	if !strings.Contains(env.LastError, "apply boom") {
		t.Fatalf("expected LastError to contain apply failure, got %q", env.LastError)
	}
	if !driver.destroyCalled {
		t.Fatal("expected destroy to be called after apply failure")
	}
	if !driver.cleanupCalled {
		t.Fatal("expected cleanup to be called after apply failure")
	}
}

func TestUpArtifactSaveFailureRollsBackAndPersistsError(t *testing.T) {
	middlewareName := registerTestMiddleware(t)
	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	driver := &fakeDriver{
		workspaceRoot: workspaceRoot,
	}
	registry, err := runtime.NewRegistry(driver)
	if err != nil {
		t.Fatalf("new runtime registry: %v", err)
	}

	envStore := newMemoryEnvironmentStore()
	envStore.saveRuntimeArtifactErr = errors.New("artifact save boom")
	coord := coordinator.New(
		fakeBlueprintStore{blueprint: testBlueprint(middlewareName)},
		envStore,
		registry,
	)

	_, err = coord.Up(context.Background(), coordinator.UpRequest{
		BlueprintFile:   "ignored.yaml",
		EnvironmentType: runtime.EnvironmentTypeCompose,
	})
	if err == nil {
		t.Fatal("expected up failure")
	}
	if !strings.Contains(err.Error(), "artifact save boom") {
		t.Fatalf("expected artifact save error, got %v", err)
	}

	env, getErr := envStore.Get(driver.environmentID)
	if getErr != nil {
		t.Fatalf("get persisted environment: %v", getErr)
	}
	if env.Status != model.EnvironmentStatusError {
		t.Fatalf("expected error status, got %s", env.Status)
	}
	if !strings.Contains(env.LastError, "artifact save boom") {
		t.Fatalf("expected LastError to contain artifact save failure, got %q", env.LastError)
	}
	if !driver.destroyCalled || !driver.cleanupCalled {
		t.Fatalf("expected destroy and cleanup after artifact save failure, got destroy=%t cleanup=%t", driver.destroyCalled, driver.cleanupCalled)
	}
}

func TestUpApplyFailureIncludesCleanupFailureInLastError(t *testing.T) {
	middlewareName := registerTestMiddleware(t)
	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	driver := &fakeDriver{
		workspaceRoot: workspaceRoot,
		applyErr:      errors.New("apply boom"),
		cleanupErr:    errors.New("cleanup boom"),
	}
	registry, err := runtime.NewRegistry(driver)
	if err != nil {
		t.Fatalf("new runtime registry: %v", err)
	}

	envStore := newMemoryEnvironmentStore()
	coord := coordinator.New(
		fakeBlueprintStore{blueprint: testBlueprint(middlewareName)},
		envStore,
		registry,
	)

	_, err = coord.Up(context.Background(), coordinator.UpRequest{
		BlueprintFile:   "ignored.yaml",
		EnvironmentType: runtime.EnvironmentTypeCompose,
	})
	if err == nil {
		t.Fatal("expected up failure")
	}
	if !strings.Contains(err.Error(), "cleanup failed: cleanup boom") {
		t.Fatalf("expected combined cleanup error, got %v", err)
	}

	env, getErr := envStore.Get(driver.environmentID)
	if getErr != nil {
		t.Fatalf("get persisted environment: %v", getErr)
	}
	if !strings.Contains(env.LastError, "cleanup failed: cleanup boom") {
		t.Fatalf("expected LastError to contain cleanup failure, got %q", env.LastError)
	}
}

type fakeBlueprintStore struct {
	blueprint model.Blueprint
}

func (f fakeBlueprintStore) LoadBlueprint(_ string) (model.Blueprint, error) {
	return f.blueprint, nil
}

func (f fakeBlueprintStore) ListBlueprints(_ string) ([]store.BlueprintFile, error) {
	return []store.BlueprintFile{{
		Path:      "ignored.yaml",
		Blueprint: f.blueprint,
	}}, nil
}

func (f fakeBlueprintStore) ResolveBlueprint(ref string, _ string) (store.BlueprintFile, error) {
	return store.BlueprintFile{
		Path:      ref,
		Blueprint: f.blueprint,
	}, nil
}

func (fakeBlueprintStore) SaveBlueprint(string, model.Blueprint) error {
	return nil
}

func (fakeBlueprintStore) UpdateBlueprint(string, model.Blueprint) error {
	return nil
}

func (fakeBlueprintStore) DeleteBlueprint(string) error {
	return nil
}

type memoryEnvironmentStore struct {
	environments           map[string]model.Environment
	artifacts              map[string]runtime.RuntimeArtifact
	saveErr                error
	saveRuntimeArtifactErr error
}

func newMemoryEnvironmentStore() *memoryEnvironmentStore {
	return &memoryEnvironmentStore{
		environments: make(map[string]model.Environment),
		artifacts:    make(map[string]runtime.RuntimeArtifact),
	}
}

func (s *memoryEnvironmentStore) Save(env model.Environment) error {
	if s.saveErr != nil {
		return s.saveErr
	}
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
	if s.saveRuntimeArtifactErr != nil {
		return s.saveRuntimeArtifactErr
	}
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

type fakeDriver struct {
	workspaceRoot string
	environmentID string
	renderErr     error
	applyErr      error
	destroyErr    error
	cleanupErr    error
	destroyCalled bool
	cleanupCalled bool
}

func (d *fakeDriver) Type() runtime.EnvironmentType {
	return runtime.EnvironmentTypeCompose
}

func (d *fakeDriver) Prepare(_ context.Context, req runtime.PrepareRequest) (*runtime.PreparePlan, error) {
	if d.environmentID == "" {
		d.environmentID = "env-" + strings.ReplaceAll(req.Input.BlueprintName, " ", "-")
	}
	workspaceDir := filepath.Join(d.workspaceRoot, d.environmentID)
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return nil, err
	}
	return &runtime.PreparePlan{
		Environment: model.Environment{
			ID:               d.environmentID,
			Name:             req.Input.BlueprintName,
			BlueprintName:    req.Input.BlueprintName,
			BlueprintVersion: req.Input.BlueprintVersion,
			RuntimeType:      string(req.Input.RuntimeType),
			Status:           model.EnvironmentStatusCreating,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		Layout: model.RuntimeLayout{
			RootDir:     workspaceDir,
			ComposeFile: filepath.Join(workspaceDir, "docker-compose.yml"),
		},
		Files: map[string]string{
			"compose_file": filepath.Join(workspaceDir, "docker-compose.yml"),
		},
		ProjectName: "project-" + d.environmentID,
	}, nil
}

func (d *fakeDriver) Render(_ context.Context, req runtime.RenderRequest) (*runtime.RenderPlan, error) {
	if d.renderErr != nil {
		return nil, d.renderErr
	}
	return &runtime.RenderPlan{
		Prepared:    req.Prepared,
		PrimaryFile: req.Prepared.Layout.ComposeFile,
		BuildScript: filepath.Join(req.Prepared.Layout.RootDir, "build.sh"),
		CheckScript: filepath.Join(req.Prepared.Layout.RootDir, "check.sh"),
	}, nil
}

func (d *fakeDriver) PlanApply(_ context.Context, req runtime.BuildApplyRequest) (*runtime.ApplyPlan, error) {
	return &runtime.ApplyPlan{
		Environment:  req.Prepared.Environment,
		WorkspaceDir: req.Prepared.Layout.RootDir,
		ProjectName:  req.Prepared.ProjectName,
		PrimaryFile:  req.Rendered.PrimaryFile,
		BuildScript:  req.Rendered.BuildScript,
		CheckScript:  req.Rendered.CheckScript,
		Services:     []string{"service-1"},
	}, nil
}

func (d *fakeDriver) Apply(_ context.Context, _ runtime.ApplyPlan) (*runtime.OperationResult, error) {
	if d.applyErr != nil {
		return nil, d.applyErr
	}
	return &runtime.OperationResult{
		Message: "apply ok",
		Changed: true,
	}, nil
}

func (d *fakeDriver) Create(_ context.Context, _ runtime.ApplyPlan) (*runtime.OperationResult, error) {
	if d.applyErr != nil {
		return nil, d.applyErr
	}
	return &runtime.OperationResult{
		Message: "create ok",
		Changed: true,
	}, nil
}

func (d *fakeDriver) PlanLifecycle(_ context.Context, req runtime.BuildLifecycleRequest) (*runtime.LifecyclePlan, error) {
	return &runtime.LifecyclePlan{
		Environment:  req.Environment,
		WorkspaceDir: req.Artifact.WorkspaceDir,
		ProjectName:  req.Artifact.ProjectName,
		PrimaryFile:  req.Artifact.PrimaryFile,
	}, nil
}

func (d *fakeDriver) Status(_ context.Context, _ runtime.LifecyclePlan) (*runtime.StatusResult, error) {
	return &runtime.StatusResult{Status: model.EnvironmentStatusRunning, Message: "ok"}, nil
}

func (d *fakeDriver) Doctor(_ context.Context, _ runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return &runtime.OperationResult{Message: "ok"}, nil
}

func (d *fakeDriver) Start(_ context.Context, _ runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return &runtime.OperationResult{Message: "ok"}, nil
}

func (d *fakeDriver) Stop(_ context.Context, _ runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return &runtime.OperationResult{Message: "ok"}, nil
}

func (d *fakeDriver) Destroy(_ context.Context, _ runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.destroyCalled = true
	if d.destroyErr != nil {
		return nil, d.destroyErr
	}
	return &runtime.OperationResult{Message: "destroy ok", Changed: true}, nil
}

func (d *fakeDriver) Cleanup(_ context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	d.cleanupCalled = true
	if d.cleanupErr != nil {
		return nil, d.cleanupErr
	}
	if plan.WorkspaceDir != "" {
		_ = os.RemoveAll(plan.WorkspaceDir)
	}
	return &runtime.OperationResult{Message: "cleanup ok", Changed: true}, nil
}

type testMiddleware struct {
	middleware string
	template   string
	services   []model.BlueprintService
}

func (m *testMiddleware) Middleware() string { return m.middleware }
func (m *testMiddleware) Template() string   { return m.template }
func (m *testMiddleware) IsDefault() bool    { return true }

func (m *testMiddleware) Configure(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	name := input.Name
	if name == "" {
		name = tpl.DefaultServiceName(m.middleware, index)
	}
	service := model.BlueprintService{
		Name:       name,
		Middleware: m.middleware,
		Template:   m.template,
		Values:     map[string]any{},
	}
	m.services = append(m.services, service)
	return service, nil
}

func (m *testMiddleware) BuildRuntimeContexts(runtimeType runtime.EnvironmentType) ([]runtime.EnvironmentContext, error) {
	contexts := make([]runtime.EnvironmentContext, 0, len(m.services))
	for _, service := range m.services {
		contexts = append(contexts, runtime.ComposeContext{
			EnvType:     runtimeType,
			ServiceName: service.Name,
			Middleware:  service.Middleware,
			Template:    service.Template,
			Service: runtime.ServiceSpec{
				Image: "busybox:latest",
			},
		})
	}
	return contexts, nil
}

func registerTestMiddleware(t *testing.T) string {
	t.Helper()

	name := "test-" + sanitizeName(t.Name())
	spec := &testMiddleware{
		middleware: name,
		template:   "single",
	}
	if err := tpl.RegisterMiddleware(
		tpl.NewMiddlewareRuntimeKey(name, "single", runtime.EnvironmentTypeCompose),
		spec,
	); err != nil {
		t.Fatalf("register test middleware: %v", err)
	}
	return name
}

func testBlueprint(middleware string) model.Blueprint {
	return model.Blueprint{
		Name:    "failure-test",
		Version: "1.0.0",
		Services: []model.BlueprintService{
			{
				Name:       "service-1",
				Middleware: middleware,
				Template:   "single",
				Values:     map[string]any{},
			},
		},
	}
}

func sanitizeName(value string) string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer("/", "-", "_", "-", " ", "-")
	return replacer.Replace(value)
}

var _ store.BlueprintStore = fakeBlueprintStore{}
var _ environment.Store = (*memoryEnvironmentStore)(nil)
var _ runtime.Driver = (*fakeDriver)(nil)
