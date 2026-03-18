package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

func TestApplyBuildsComposeCommandAndExtractsEndpoints(t *testing.T) {
	runner := &fakeRunner{
		outputs: []fakeRunResult{
			{output: "created"},
			{output: `[{"Service":"mock-1","State":"running","Publishers":[{"URL":"127.0.0.1","PublishedPort":3306,"Protocol":"tcp"}]}]`},
		},
	}
	executor := NewExecutor("", runner)
	env, workspaceDir := testEnvironment(t)
	plan := runtime.ApplyPlan{
		Environment:  env,
		WorkspaceDir: workspaceDir,
		ProjectName:  "demo-project",
		PrimaryFile:  filepath.Join(workspaceDir, "docker-compose.yaml"),
		BuildScript:  filepath.Join(workspaceDir, "build.sh"),
	}
	if err := os.WriteFile(plan.BuildScript, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write build script: %v", err)
	}

	result, err := executor.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !result.Changed {
		t.Fatal("expected apply to report changed")
	}
	if len(result.Endpoints) != 1 {
		t.Fatalf("expected one endpoint, got %d", len(result.Endpoints))
	}
	if got := runner.calls[0].name; got != "/bin/sh" {
		t.Fatalf("unexpected apply command: %s", got)
	}
	if got := runner.calls[0].args; fmt.Sprint(got) != "["+plan.BuildScript+"]" {
		t.Fatalf("unexpected apply args: %v", got)
	}
	if got := runner.calls[1].args; fmt.Sprint(got) != "[compose -p demo-project -f "+plan.PrimaryFile+" ps -a --format json]" {
		t.Fatalf("unexpected status args: %v", got)
	}
}

func TestStatusMapsStoppedState(t *testing.T) {
	runner := &fakeRunner{
		outputs: []fakeRunResult{
			{output: `[{"Service":"mock-1","State":"exited","Publishers":null}]`},
		},
	}
	executor := NewExecutor("", runner)

	result, err := executor.Status(context.Background(), testLifecyclePlan(t))
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if result.Status != model.EnvironmentStatusStopped {
		t.Fatalf("expected stopped, got %s", result.Status)
	}
}

func TestDoctorExecutesCheckScript(t *testing.T) {
	runner := &fakeRunner{
		outputs: []fakeRunResult{
			{output: "mysql check ok"},
		},
	}
	executor := NewExecutor("", runner)
	plan := testLifecyclePlan(t)
	checkScript := filepath.Join(plan.WorkspaceDir, "check.sh")
	if err := os.WriteFile(checkScript, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write check script: %v", err)
	}

	result, err := executor.Doctor(context.Background(), plan)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if result.Changed {
		t.Fatal("expected doctor to report unchanged")
	}
	if got := runner.calls[0].name; got != "/bin/sh" {
		t.Fatalf("unexpected doctor command: %s", got)
	}
	if got := runner.calls[0].args; fmt.Sprint(got) != "["+checkScript+"]" {
		t.Fatalf("unexpected doctor args: %v", got)
	}
}

func TestCleanupRemovesWorkspaceDir(t *testing.T) {
	executor := NewExecutor("", &fakeRunner{})
	env, workspaceDir := testEnvironment(t)
	plan := runtime.LifecyclePlan{
		Environment:  env,
		WorkspaceDir: workspaceDir,
		ProjectName:  "demo-project",
		PrimaryFile:  filepath.Join(workspaceDir, "docker-compose.yaml"),
	}

	if err := os.WriteFile(filepath.Join(workspaceDir, "tmp.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := os.Stat(workspaceDir); err != nil {
		t.Fatalf("stat workspace: %v", err)
	}

	if _, err := executor.Cleanup(context.Background(), plan); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if _, err := os.Stat(workspaceDir); !os.IsNotExist(err) {
		t.Fatalf("expected workspace dir removed, got err=%v", err)
	}
}

type fakeRunner struct {
	calls   []fakeCall
	outputs []fakeRunResult
}

type fakeCall struct {
	workdir string
	name    string
	args    []string
}

type fakeRunResult struct {
	output string
	err    error
}

func (f *fakeRunner) Run(_ context.Context, workdir string, name string, args ...string) (string, error) {
	f.calls = append(f.calls, fakeCall{
		workdir: workdir,
		name:    name,
		args:    append([]string(nil), args...),
	})
	if len(f.outputs) == 0 {
		return "", nil
	}
	result := f.outputs[0]
	f.outputs = f.outputs[1:]
	return result.output, result.err
}

func testEnvironment(t *testing.T) (model.Environment, string) {
	t.Helper()
	workspaceDir := t.TempDir()
	composeFile := filepath.Join(workspaceDir, "docker-compose.yaml")
	if err := os.WriteFile(composeFile, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	return model.Environment{
		Name: "demo",
	}, workspaceDir
}

func testLifecyclePlan(t *testing.T) runtime.LifecyclePlan {
	t.Helper()
	env, workspaceDir := testEnvironment(t)
	return runtime.LifecyclePlan{
		Environment:  env,
		WorkspaceDir: workspaceDir,
		ProjectName:  "demo-project",
		PrimaryFile:  filepath.Join(workspaceDir, "docker-compose.yaml"),
	}
}
