package create

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuenqlve/zygarde/internal/app"
	"github.com/xuenqlve/zygarde/internal/config"
	"github.com/xuenqlve/zygarde/internal/environment"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

var requiredRuntimeFiles = []string{"compose_file", "env_file", "build_script", "check_script", "readme_file"}

type lifecycleTestContext struct {
	t        *testing.T
	dir      string
	app      *app.App
	envStore environment.FileStore
}

type lifecycleUpResult struct {
	EnvironmentID string
	WorkspaceDir  string
	ProjectName   string
}

func newLifecycleTestContext(t *testing.T) *lifecycleTestContext {
	t.Helper()

	if testing.Short() {
		t.Skip("skip integration test in short mode")
	}
	if os.Getenv("ZYGARDE_CONTAINER_ENGINE") == "" {
		t.Setenv("ZYGARDE_CONTAINER_ENGINE", "podman")
	}
	requireContainerReady(t)
	t.Logf("container engine: %s", config.Default().ContainerEngine)

	dir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(previousDir); chdirErr != nil {
			t.Fatalf("restore working directory: %v", chdirErr)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Logf("test workspace: %s", dir)

	application, err := app.New()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	return &lifecycleTestContext{
		t:        t,
		dir:      dir,
		app:      application,
		envStore: environment.NewFileStore(".zygarde/environments"),
	}
}

func (tc *lifecycleTestContext) writeBlueprint(content string) string {
	tc.t.Helper()

	path := filepath.Join(tc.dir, "zygarde.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tc.t.Fatalf("write blueprint: %v", err)
	}
	tc.t.Logf("blueprint path: %s", path)
	return path
}

func (tc *lifecycleTestContext) up(blueprintPath string) *lifecycleUpResult {
	tc.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	tc.t.Cleanup(cancel)

	var cleanupTarget string
	tc.t.Cleanup(func() {
		if cleanupTarget == "" {
			return
		}
		tc.t.Logf("cleanup fallback: down environment %s", cleanupTarget)
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cleanupCancel()
		_, _ = tc.app.Down(cleanupCtx, cleanupTarget)
	})

	tc.t.Log("step up: create runtime assets and start environment")
	result, err := tc.app.Up(ctx, blueprintPath, runtime.EnvironmentTypeCompose)
	if err != nil {
		tc.t.Fatalf("up environment: %v", err)
	}

	upResult := &lifecycleUpResult{
		EnvironmentID: result.EnvironmentID,
		WorkspaceDir:  result.WorkspaceDir,
		ProjectName:   result.ProjectName,
	}
	if upResult.EnvironmentID == "" {
		tc.t.Fatal("expected environment id after up")
	}
	if upResult.WorkspaceDir == "" {
		tc.t.Fatal("expected workspace dir after up")
	}
	if upResult.ProjectName == "" {
		tc.t.Fatal("expected project name after up")
	}
	cleanupTarget = upResult.EnvironmentID

	tc.t.Logf("up result: environment_id=%s workspace_dir=%s project_name=%s", upResult.EnvironmentID, upResult.WorkspaceDir, upResult.ProjectName)
	tc.t.Logf("up message: %s", result.Message)
	return upResult
}

func (tc *lifecycleTestContext) verifyCurrentEnvironment(upResult *lifecycleUpResult) environment.CurrentEnvironment {
	tc.t.Helper()

	current, err := environment.LoadCurrent()
	if err != nil {
		tc.t.Fatalf("load current environment marker: %v", err)
	}
	if current.EnvironmentID != upResult.EnvironmentID {
		tc.t.Fatalf("unexpected current environment id: got=%s want=%s", current.EnvironmentID, upResult.EnvironmentID)
	}
	if current.WorkspaceDir != upResult.WorkspaceDir {
		tc.t.Fatalf("unexpected current workspace dir: got=%s want=%s", current.WorkspaceDir, upResult.WorkspaceDir)
	}
	tc.t.Logf("current environment marker: %+v", current)
	return current
}

func (tc *lifecycleTestContext) verifyRunningEnvironment(upResult *lifecycleUpResult) (model.Environment, runtime.RuntimeArtifact) {
	tc.t.Helper()

	env, err := tc.envStore.Get(upResult.EnvironmentID)
	if err != nil {
		tc.t.Fatalf("load environment record: %v", err)
	}
	if env.Status != model.EnvironmentStatusRunning {
		tc.t.Fatalf("expected running status after up, got %s", env.Status)
	}
	tc.t.Logf("environment record after up: id=%s status=%s created_at=%s updated_at=%s", env.ID, env.Status, env.CreatedAt.Format(time.RFC3339), env.UpdatedAt.Format(time.RFC3339))

	artifact, err := tc.envStore.GetRuntimeArtifact(upResult.EnvironmentID)
	if err != nil {
		tc.t.Fatalf("load runtime artifact: %v", err)
	}
	tc.t.Logf("runtime artifact: environment_id=%s workspace_dir=%s primary_file=%s project_name=%s", artifact.EnvironmentID, artifact.WorkspaceDir, artifact.PrimaryFile, artifact.ProjectName)
	return env, artifact
}

func (tc *lifecycleTestContext) verifyRuntimeFiles(artifact runtime.RuntimeArtifact) {
	tc.t.Helper()

	requireFileExists(tc.t, artifact.PrimaryFile)
	for _, key := range requiredRuntimeFiles {
		path, ok := artifact.Files[key]
		if !ok || path == "" {
			tc.t.Fatalf("expected runtime artifact file %s", key)
		}
		requireFileExists(tc.t, path)
		tc.t.Logf("runtime file verified: %s => %s", key, path)
	}
}

func (tc *lifecycleTestContext) verifyStatusRunning() {
	tc.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tc.t.Log("step status: verify current directory environment is running")
	result, err := tc.app.Status(ctx, "")
	if err != nil {
		tc.t.Fatalf("status current environment: %v", err)
	}
	if !strings.Contains(result.Message, "running") {
		tc.t.Fatalf("expected running status message, got %q", result.Message)
	}
	tc.t.Logf("status result: %s", result.Message)
}

func (tc *lifecycleTestContext) waitForDoctorPass(timeout time.Duration) {
	tc.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	tc.t.Log("step doctor: wait until generated check.sh passes")
	for {
		result, err := tc.app.Doctor(ctx, "")
		if err == nil {
			if !strings.Contains(result.Message, "doctor passed") {
				tc.t.Fatalf("unexpected doctor message: %q", result.Message)
			}
			tc.t.Logf("doctor result: %s", result.Message)
			return
		}
		tc.t.Logf("doctor retry: %v", err)

		select {
		case <-ctx.Done():
			tc.t.Fatalf("doctor did not pass before timeout: %v", err)
		case <-ticker.C:
		}
	}
}

func (tc *lifecycleTestContext) downAndVerify(upResult *lifecycleUpResult) {
	tc.t.Helper()

	downCtx, downCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer downCancel()

	tc.t.Log("step down: stop and remove current directory environment")
	result, err := tc.app.Down(downCtx, "")
	if err != nil {
		tc.t.Fatalf("down current environment: %v", err)
	}
	tc.t.Logf("down result: %s", result.Message)

	env, err := tc.envStore.Get(upResult.EnvironmentID)
	if err != nil {
		tc.t.Fatalf("reload environment record after down: %v", err)
	}
	if env.Status != model.EnvironmentStatusDestroyed {
		tc.t.Fatalf("expected destroyed status after down, got %s", env.Status)
	}
	tc.t.Logf("environment record after down: id=%s status=%s updated_at=%s", env.ID, env.Status, env.UpdatedAt.Format(time.RFC3339))

	if _, err := os.Stat(upResult.WorkspaceDir); !os.IsNotExist(err) {
		tc.t.Fatalf("expected workspace removed after down, got err=%v", err)
	}
	tc.t.Logf("workspace removed: %s", upResult.WorkspaceDir)

	if _, err := environment.LoadCurrent(); err == nil {
		tc.t.Fatal("expected current environment marker removed after down")
	}
	tc.t.Log("current environment marker removed")
}

func requireContainerReady(t *testing.T) {
	t.Helper()

	containerEngine := config.Default().ContainerEngine
	if _, err := exec.LookPath(containerEngine); err != nil {
		t.Skipf("%s is required: %v", containerEngine, err)
	}
	output, err := exec.Command(containerEngine, "compose", "version").CombinedOutput()
	if err != nil {
		t.Skipf("%s compose is required: %v, output: %s", containerEngine, err, strings.TrimSpace(string(output)))
	}
}

func requireFileExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("expected file, got directory: %s", path)
	}
}

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func localImageOrSkip(t *testing.T, defaultImage string, candidates ...string) string {
	t.Helper()

	containerEngine := config.Default().ContainerEngine
	if containerEngine == "" {
		return defaultImage
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if imageExists(containerEngine, candidate) {
			t.Logf("use local image: %s", candidate)
			return candidate
		}
	}

	if containerEngine == "podman" {
		t.Skipf("skip integration test: no local image available for %s; checked %s", containerEngine, strings.Join(candidates, ", "))
	}
	return defaultImage
}

func imageExists(containerEngine, image string) bool {
	cmd := exec.Command(containerEngine, "image", "exists", image)
	return cmd.Run() == nil
}
