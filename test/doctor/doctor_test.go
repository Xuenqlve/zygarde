package doctor_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	compose "github.com/xuenqlve/zygarde/internal/deployment/compose"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

func TestComposeDoctorReturnsHelpfulErrorWhenCheckScriptMissing(t *testing.T) {
	executor := compose.NewExecutor("", &fakeRunner{})
	plan := testLifecyclePlan(t)

	_, err := executor.Doctor(context.Background(), plan)
	if err == nil {
		t.Fatal("expected missing check.sh error")
	}
	if !strings.Contains(err.Error(), "compose doctor") {
		t.Fatalf("expected compose doctor prefix, got %v", err)
	}
	if !strings.Contains(err.Error(), "check.sh") {
		t.Fatalf("expected check.sh path detail, got %v", err)
	}
}

func TestComposeDoctorReturnsRunnerOutputWhenCheckScriptFails(t *testing.T) {
	runner := &fakeRunner{
		output: "mysql not ready yet",
		err:    errors.New("exit status 1"),
	}
	executor := compose.NewExecutor("", runner)
	plan := testLifecyclePlan(t)
	checkScript := filepath.Join(plan.WorkspaceDir, "check.sh")
	if err := os.WriteFile(checkScript, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write check script: %v", err)
	}

	_, err := executor.Doctor(context.Background(), plan)
	if err == nil {
		t.Fatal("expected doctor failure")
	}
	if !strings.Contains(err.Error(), "compose doctor: exit status 1: mysql not ready yet") {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.name != "/bin/sh" {
		t.Fatalf("expected /bin/sh runner, got %q", runner.name)
	}
	if len(runner.args) != 1 || runner.args[0] != checkScript {
		t.Fatalf("unexpected runner args: %v", runner.args)
	}
}

func TestComposeDoctorTrimsWhitespaceFromRunnerOutputOnFailure(t *testing.T) {
	runner := &fakeRunner{
		output: "\n  service unhealthy  \n",
		err:    errors.New("exit status 2"),
	}
	executor := compose.NewExecutor("", runner)
	plan := testLifecyclePlan(t)
	checkScript := filepath.Join(plan.WorkspaceDir, "check.sh")
	if err := os.WriteFile(checkScript, []byte("#!/bin/sh\nexit 2\n"), 0o755); err != nil {
		t.Fatalf("write check script: %v", err)
	}

	_, err := executor.Doctor(context.Background(), plan)
	if err == nil {
		t.Fatal("expected doctor failure")
	}
	if !strings.Contains(err.Error(), "service unhealthy") {
		t.Fatalf("expected trimmed output in error, got %v", err)
	}
	if strings.Contains(err.Error(), "\n  service unhealthy  \n") {
		t.Fatalf("expected output to be trimmed, got %v", err)
	}
}

type fakeRunner struct {
	workdir string
	name    string
	args    []string
	output  string
	err     error
}

func (f *fakeRunner) Run(_ context.Context, workdir string, name string, args ...string) (string, error) {
	f.workdir = workdir
	f.name = name
	f.args = append([]string(nil), args...)
	return f.output, f.err
}

func testLifecyclePlan(t *testing.T) runtime.LifecyclePlan {
	t.Helper()

	workspaceDir := t.TempDir()
	composeFile := filepath.Join(workspaceDir, "docker-compose.yaml")
	if err := os.WriteFile(composeFile, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	return runtime.LifecyclePlan{
		Environment: model.Environment{
			Name: "demo",
		},
		WorkspaceDir: workspaceDir,
		ProjectName:  "demo-project",
		PrimaryFile:  composeFile,
	}
}
