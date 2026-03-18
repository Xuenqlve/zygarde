package compose

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuenqlve/zygarde/internal/config"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

func TestExecutorWithMySQLSingleV57(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test in short mode")
	}
	containerEngine := config.Default().ContainerEngine
	if _, err := exec.LookPath(containerEngine); err != nil {
		t.Skipf("%s is required: %v", containerEngine, err)
	}
	if output, err := exec.Command(containerEngine, "compose", "version").CombinedOutput(); err != nil {
		t.Skipf("%s compose is required: %v, output: %s", containerEngine, err, strings.TrimSpace(string(output)))
	}

	projectName := fmt.Sprintf("zygarde-it-%d", time.Now().UnixNano())
	containerName := projectName + "-mysql"
	mysqlPort := freePort(t)
	rootPassword := "root123"

	workspaceDir := t.TempDir()
	composeFile, buildScript := prepareMySQLSingleFixture(t, workspaceDir, containerName, mysqlPort, rootPassword)
	env := model.Environment{
		ID:   projectName,
		Name: "mysql-single-v57",
	}
	lifecyclePlan := runtime.LifecyclePlan{
		Environment:  env,
		WorkspaceDir: workspaceDir,
		ProjectName:  projectName,
		PrimaryFile:  composeFile,
	}
	executor := NewExecutor(containerEngine, nil)

	t.Logf("fixture: mysql single v5.7")
	t.Logf("workspace: %s", workspaceDir)
	t.Logf("compose file: %s", composeFile)
	t.Logf("build script: %s", buildScript)
	t.Logf("compose project: %s", projectName)
	t.Logf("container name: %s", containerName)
	t.Logf("mysql port: %d", mysqlPort)

	t.Cleanup(func() {
		t.Log("cleanup: execute compose down and remove workspace")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		_, _ = executor.Destroy(ctx, lifecyclePlan)
		_, _ = executor.Cleanup(ctx, lifecyclePlan)
	})

	applyCtx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	t.Log("step apply: execute compose up -d")
	applyResult, err := executor.Apply(applyCtx, runtime.ApplyPlan{
		Environment:  env,
		WorkspaceDir: workspaceDir,
		ProjectName:  projectName,
		PrimaryFile:  composeFile,
		BuildScript:  buildScript,
		Services:     []string{"mysql"},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !applyResult.Changed {
		t.Fatal("expected apply to report changed")
	}
	t.Logf("apply result: %s", applyResult.Message)

	t.Log("verify apply: use executor.Status -> compose ps -a --format json and expect running")
	statusResult := waitForStatus(t, applyCtx, executor, lifecyclePlan, model.EnvironmentStatusRunning, 3*time.Second)
	if len(statusResult.Endpoints) == 0 {
		t.Fatal("expected endpoints after apply")
	}
	t.Logf("status after apply: %s", statusResult.Message)
	t.Logf("endpoints after apply: %+v", statusResult.Endpoints)

	t.Log("verify mysql connectivity: execute container engine exec <container> mysql -uroot -p... -e 'SELECT 1;'")
	if err := waitForMySQLQuery(applyCtx, containerEngine, containerName, rootPassword); err != nil {
		t.Fatalf("mysql connectivity check failed: %v", err)
	}
	t.Log("mysql connectivity check passed")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer stopCancel()

	t.Log("step stop: execute compose stop")
	stopResult, err := executor.Stop(stopCtx, lifecyclePlan)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if !stopResult.Changed {
		t.Fatal("expected stop to report changed")
	}
	t.Logf("stop result: %s", stopResult.Message)
	t.Log("verify stop: use executor.Status -> compose ps -a --format json and expect stopped")
	stoppedStatus := waitForStatus(t, stopCtx, executor, lifecyclePlan, model.EnvironmentStatusStopped, 2*time.Second)
	t.Logf("status after stop: %s", stoppedStatus.Message)

	destroyCtx, destroyCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer destroyCancel()

	t.Log("step destroy: execute compose down")
	destroyResult, err := executor.Destroy(destroyCtx, lifecyclePlan)
	if err != nil {
		t.Fatalf("destroy: %v", err)
	}
	if !destroyResult.Changed {
		t.Fatal("expected destroy to report changed")
	}
	t.Logf("destroy result: %s", destroyResult.Message)
	t.Log("verify destroy: use executor.Status -> compose ps -a --format json and expect destroyed")
	destroyedStatus := waitForStatus(t, destroyCtx, executor, lifecyclePlan, model.EnvironmentStatusDestroyed, 2*time.Second)
	t.Logf("status after destroy: %s", destroyedStatus.Message)

	t.Log("step cleanup: remove runtime workspace directory")
	if _, err := executor.Cleanup(destroyCtx, lifecyclePlan); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if _, err := os.Stat(workspaceDir); !os.IsNotExist(err) {
		t.Fatalf("expected workspace removed after cleanup, got err=%v", err)
	}
	t.Log("cleanup verification passed: workspace directory removed")
}

func prepareMySQLSingleFixture(t *testing.T, workspaceDir, containerName string, mysqlPort int, rootPassword string) (string, string) {
	t.Helper()

	sourceDir := filepath.Join("..", "..", "..", "docker", "mysql", "single_v5.7")
	composeBytes, err := os.ReadFile(filepath.Join(sourceDir, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("read fixture compose file: %v", err)
	}

	composeContent := string(composeBytes)
	composeContent = strings.Replace(
		composeContent,
		"    image: mysql:5.7\n",
		"    image: mysql:5.7\n    platform: linux/amd64\n",
		1,
	)
	composeContent = strings.ReplaceAll(composeContent, "container_name: zygarde-mysql-single", "container_name: "+containerName)
	composeContent = strings.ReplaceAll(composeContent, "data/mysql", ".testdata/mysql")

	composePath := filepath.Join(workspaceDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0o644); err != nil {
		t.Fatalf("write compose fixture: %v", err)
	}

	envContent := fmt.Sprintf("MYSQL_VERSION=mysql:5.7\nMYSQL_PORT=%d\nMYSQL_ROOT_PASSWORD=%s\n", mysqlPort, rootPassword)
	if err := os.WriteFile(filepath.Join(workspaceDir, ".env"), []byte(envContent), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	buildScript := filepath.Join(workspaceDir, "build.sh")
	buildContent := fmt.Sprintf("#!/usr/bin/env bash\nset -euo pipefail\nROOT_DIR=\"$(cd \"$(dirname \"$0\")\" && pwd)\"\ncd \"$ROOT_DIR\"\nCONTAINER_ENGINE=\"${ZYGARDE_CONTAINER_ENGINE:-%s}\"\nif [ -f .env ]; then\n    set -a\n    . ./.env\n    set +a\nfi\n\n\"$CONTAINER_ENGINE\" compose -p %q -f docker-compose.yml up -d\n", config.Default().ContainerEngine, containerName[:len(containerName)-len("-mysql")])
	if err := os.WriteFile(buildScript, []byte(buildContent), 0o755); err != nil {
		t.Fatalf("write build script: %v", err)
	}

	return composePath, buildScript
}

func waitForStatus(t *testing.T, ctx context.Context, executor Executor, plan runtime.LifecyclePlan, expected model.EnvironmentStatus, interval time.Duration) *runtime.StatusResult {
	t.Helper()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		result, err := executor.Status(ctx, plan)
		if err == nil && result.Status == expected {
			return result
		}
		if err == nil {
			t.Logf("wait status: current=%s expected=%s message=%s", result.Status, expected, result.Message)
		}

		select {
		case <-ctx.Done():
			if err != nil {
				t.Fatalf("status wait failed: %v", err)
			}
			t.Fatalf("status wait timed out, expected %s", expected)
		case <-ticker.C:
		}
	}
}

func waitForMySQLQuery(ctx context.Context, containerEngine, containerName, rootPassword string) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		cmd := exec.CommandContext(
			ctx,
			containerEngine,
			"exec",
			containerName,
			"mysql",
			"-uroot",
			"-p"+rootPassword,
			"-e",
			"SELECT 1;",
		)
		if output, err := cmd.CombinedOutput(); err == nil {
			if strings.Contains(string(output), "1") {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
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
