package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuenqlve/zygarde/internal/model"
)

func TestApplyBuildsComposeCommandAndExtractsEndpoints(t *testing.T) {
	runner := &fakeRunner{
		outputs: []fakeRunResult{
			{output: "created"},
			{output: `[{"Service":"mock-1","State":"running","Publishers":[{"URL":"127.0.0.1","PublishedPort":3306,"Protocol":"tcp"}]}]`},
		},
	}
	executor := NewExecutor(runner)
	env := testEnvironment(t)
	rendered := model.RenderResult{PrimaryFile: env.ComposeFile}

	result, err := executor.Apply(context.Background(), env, rendered)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !result.Changed {
		t.Fatal("expected apply to report changed")
	}
	if len(result.Endpoints) != 1 {
		t.Fatalf("expected one endpoint, got %d", len(result.Endpoints))
	}
	if got := runner.calls[0].args; fmt.Sprint(got) != "[compose -p demo-project -f "+env.ComposeFile+" up -d]" {
		t.Fatalf("unexpected apply args: %v", got)
	}
	if got := runner.calls[1].args; fmt.Sprint(got) != "[compose -p demo-project -f "+env.ComposeFile+" ps -a --format json]" {
		t.Fatalf("unexpected status args: %v", got)
	}
}

func TestStatusMapsStoppedState(t *testing.T) {
	runner := &fakeRunner{
		outputs: []fakeRunResult{
			{output: `[{"Service":"mock-1","State":"exited","Publishers":null}]`},
		},
	}
	executor := NewExecutor(runner)

	result, err := executor.Status(context.Background(), testEnvironment(t))
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if result.Status != model.EnvironmentStatusStopped {
		t.Fatalf("expected stopped, got %s", result.Status)
	}
}

func TestCleanupRemovesWorkspaceDir(t *testing.T) {
	executor := NewExecutor(&fakeRunner{})
	env := testEnvironment(t)

	if err := os.WriteFile(filepath.Join(env.WorkspaceDir, "tmp.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := os.Stat(env.WorkspaceDir); err != nil {
		t.Fatalf("stat workspace: %v", err)
	}

	if _, err := executor.Cleanup(context.Background(), env); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if _, err := os.Stat(env.WorkspaceDir); !os.IsNotExist(err) {
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

func testEnvironment(t *testing.T) model.Environment {
	t.Helper()
	workspaceDir := t.TempDir()
	composeFile := filepath.Join(workspaceDir, "docker-compose.yaml")
	if err := os.WriteFile(composeFile, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	return model.Environment{
		Name:         "demo",
		ProjectName:  "demo-project",
		WorkspaceDir: workspaceDir,
		ComposeFile:  composeFile,
	}
}
