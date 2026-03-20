package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuenqlve/zygarde/internal/cli"
	"github.com/xuenqlve/zygarde/internal/environment"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

func TestFileStoreListReturnsSortedEnvironmentsAndSkipsRuntimeArtifacts(t *testing.T) {
	store := environment.NewFileStore(t.TempDir())

	oldEnv := model.Environment{
		ID:        "env-old",
		Name:      "old",
		Status:    model.EnvironmentStatusDestroyed,
		CreatedAt: time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 18, 9, 5, 0, 0, time.UTC),
	}
	newEnv := model.Environment{
		ID:        "env-new",
		Name:      "new",
		Status:    model.EnvironmentStatusRunning,
		CreatedAt: time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 18, 10, 5, 0, 0, time.UTC),
	}
	if err := store.Save(oldEnv); err != nil {
		t.Fatalf("save old environment: %v", err)
	}
	if err := store.Save(newEnv); err != nil {
		t.Fatalf("save new environment: %v", err)
	}
	if err := store.SaveRuntimeArtifact(runtime.RuntimeArtifact{EnvironmentID: "env-old"}); err != nil {
		t.Fatalf("save runtime artifact: %v", err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("list environments: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 environments, got %d", len(got))
	}
	if got[0].ID != "env-new" {
		t.Fatalf("expected newest environment first, got %q", got[0].ID)
	}
	if got[1].ID != "env-old" {
		t.Fatalf("expected oldest environment second, got %q", got[1].ID)
	}
}

func TestFileStoreListReturnsEmptySliceWhenRootMissing(t *testing.T) {
	store := environment.NewFileStore(t.TempDir() + "/missing")

	got, err := store.List()
	if err != nil {
		t.Fatalf("list missing root: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %d items", len(got))
	}
}

func TestFileStoreListFallsBackToCreatedAtAndIDForStableSort(t *testing.T) {
	store := environment.NewFileStore(t.TempDir())

	alpha := model.Environment{
		ID:        "env-alpha",
		Name:      "alpha",
		Status:    model.EnvironmentStatusStopped,
		CreatedAt: time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC),
	}
	bravo := model.Environment{
		ID:        "env-bravo",
		Name:      "bravo",
		Status:    model.EnvironmentStatusRunning,
		CreatedAt: time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC),
	}
	charlie := model.Environment{
		ID:        "env-charlie",
		Name:      "charlie",
		Status:    model.EnvironmentStatusRunning,
		CreatedAt: time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC),
	}
	for _, env := range []model.Environment{alpha, bravo, charlie} {
		if err := store.Save(env); err != nil {
			t.Fatalf("save environment %s: %v", env.ID, err)
		}
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("list environments: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 environments, got %d", len(got))
	}
	wantOrder := []string{"env-charlie", "env-alpha", "env-bravo"}
	for i, want := range wantOrder {
		if got[i].ID != want {
			t.Fatalf("unexpected order at index %d: got=%s want=%s", i, got[i].ID, want)
		}
	}
}

func TestRunListPrintsHelpfulMessageWhenNoEnvironmentsExist(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{"list"}, &stdout); err != nil {
		t.Fatalf("run list: %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != "no environments found" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRunListPrintsPersistedEnvironments(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	if err := os.MkdirAll(filepath.Join(".zygarde", "environments"), 0o755); err != nil {
		t.Fatalf("create environment dir: %v", err)
	}

	updatedAt := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	envFile := filepath.Join(".zygarde", "environments", "env-1.json")
	content := []byte(`{
  "ID": "env-1",
  "Name": "mysql-test",
  "BlueprintName": "mysql-test",
  "RuntimeType": "compose",
  "Status": "running",
  "Endpoints": [
    {"Name":"mysql","Host":"127.0.0.1","Port":3306,"Protocol":"tcp"}
  ],
  "CreatedAt": "2026-03-18T11:00:00Z",
  "UpdatedAt": "` + updatedAt.Format(time.RFC3339) + `",
  "LastError": ""
}
`)
	if err := os.WriteFile(envFile, content, 0o644); err != nil {
		t.Fatalf("write environment file: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{"list"}, &stdout); err != nil {
		t.Fatalf("run list: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"ID",
		"NAME",
		"env-1",
		"mysql-test",
		string(model.EnvironmentStatusRunning),
		"compose",
		updatedAt.Format(time.RFC3339),
		"127.0.0.1:3306/tcp",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
}

func TestRunListPrintsNewestEnvironmentFirstAndSkipsRuntimeArtifacts(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	root := filepath.Join(".zygarde", "environments")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create environment dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "env-old.json"), []byte(`{
  "ID": "env-old",
  "Name": "old-env",
  "BlueprintName": "old-env",
  "RuntimeType": "compose",
  "Status": "stopped",
  "CreatedAt": "2026-03-18T10:00:00Z",
  "UpdatedAt": "2026-03-18T10:05:00Z"
}`), 0o644); err != nil {
		t.Fatalf("write old environment: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "env-new.json"), []byte(`{
  "ID": "env-new",
  "Name": "new-env",
  "BlueprintName": "new-env",
  "RuntimeType": "compose",
  "Status": "running",
  "CreatedAt": "2026-03-18T11:00:00Z",
  "UpdatedAt": "2026-03-18T11:05:00Z"
}`), 0o644); err != nil {
		t.Fatalf("write new environment: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "env-new.runtime.json"), []byte(`{"EnvironmentID":"env-new"}`), 0o644); err != nil {
		t.Fatalf("write runtime artifact: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{"list"}, &stdout); err != nil {
		t.Fatalf("run list: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected header plus 2 rows, got %q", stdout.String())
	}
	if !strings.Contains(lines[1], "env-new") {
		t.Fatalf("expected newest environment first, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "env-old") {
		t.Fatalf("expected oldest environment second, got %q", lines[2])
	}
	if strings.Contains(stdout.String(), "runtime.json") {
		t.Fatalf("did not expect runtime artifact file to appear in output: %q", stdout.String())
	}
}

func TestRunListReturnsErrorWhenEnvironmentJSONIsInvalid(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	root := filepath.Join(".zygarde", "environments")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create environment dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write broken environment: %v", err)
	}

	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"list"}, &stdout)
	if err == nil {
		t.Fatal("expected list error for invalid environment json")
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
