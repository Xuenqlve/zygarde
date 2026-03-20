package environment_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuenqlve/zygarde/internal/environment"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

func TestSaveLoadAndClearCurrent(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	want := environment.CurrentEnvironment{
		EnvironmentID: "env-1",
		WorkspaceDir:  "/tmp/env-1",
		ProjectName:   "zygarde-env-1",
	}
	if err := environment.SaveCurrent(want); err != nil {
		t.Fatalf("save current environment: %v", err)
	}

	got, err := environment.LoadCurrent()
	if err != nil {
		t.Fatalf("load current environment: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected current environment: got=%+v want=%+v", got, want)
	}

	if err := environment.ClearCurrent(); err != nil {
		t.Fatalf("clear current environment: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".zygarde", "current-environment")); !os.IsNotExist(err) {
		t.Fatalf("expected current environment file removed, got err=%v", err)
	}
}

func TestSaveCurrentRequiresEnvironmentID(t *testing.T) {
	err := environment.SaveCurrent(environment.CurrentEnvironment{})
	if err == nil {
		t.Fatal("expected save current error")
	}
	if err.Error() != "current environment id is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadCurrentReturnsErrorForInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	path := filepath.Join(dir, ".zygarde", "current-environment")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir current dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{invalid json"), 0o644); err != nil {
		t.Fatalf("write current marker: %v", err)
	}

	_, err := environment.LoadCurrent()
	if err == nil {
		t.Fatal("expected invalid json error")
	}
}

func TestLoadCurrentReturnsErrorWhenEnvironmentIDMissing(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	path := filepath.Join(dir, ".zygarde", "current-environment")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir current dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"workspace_dir":"/tmp/demo"}`), 0o644); err != nil {
		t.Fatalf("write current marker: %v", err)
	}

	_, err := environment.LoadCurrent()
	if err == nil {
		t.Fatal("expected missing environment id error")
	}
	if err.Error() != "current environment id is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileStoreGetReturnsErrorForInvalidEnvironmentJSON(t *testing.T) {
	dir := t.TempDir()
	store := environment.NewFileStore(dir)
	if err := os.WriteFile(filepath.Join(dir, "env-1.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write broken environment: %v", err)
	}

	_, err := store.Get("env-1")
	if err == nil {
		t.Fatal("expected invalid environment json error")
	}
}

func TestFileStoreListReturnsErrorForInvalidEnvironmentJSON(t *testing.T) {
	dir := t.TempDir()
	store := environment.NewFileStore(dir)
	if err := os.WriteFile(filepath.Join(dir, "env-1.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write broken environment: %v", err)
	}

	_, err := store.List()
	if err == nil {
		t.Fatal("expected invalid environment json error")
	}
}

func TestFileStoreGetRuntimeArtifactReturnsErrorForInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	store := environment.NewFileStore(dir)
	if err := os.WriteFile(filepath.Join(dir, "env-1.runtime.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write broken artifact: %v", err)
	}

	_, err := store.GetRuntimeArtifact("env-1")
	if err == nil {
		t.Fatal("expected invalid runtime artifact json error")
	}
}

func TestFileStoreSaveRequiresEnvironmentID(t *testing.T) {
	store := environment.NewFileStore(t.TempDir())

	err := store.Save(model.Environment{})
	if err == nil {
		t.Fatal("expected save environment error")
	}
	if err.Error() != "environment id is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileStoreSaveRuntimeArtifactRequiresEnvironmentID(t *testing.T) {
	store := environment.NewFileStore(t.TempDir())

	err := store.SaveRuntimeArtifact(runtime.RuntimeArtifact{})
	if err == nil {
		t.Fatal("expected save runtime artifact error")
	}
	if err.Error() != "runtime artifact environment id is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileStoreListSortsByUpdatedAtThenCreatedAt(t *testing.T) {
	store := environment.NewFileStore(t.TempDir())
	if err := store.Save(model.Environment{
		ID:        "env-old",
		Name:      "old",
		Status:    model.EnvironmentStatusDestroyed,
		CreatedAt: time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 18, 9, 5, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("save old environment: %v", err)
	}
	if err := store.Save(model.Environment{
		ID:        "env-new",
		Name:      "new",
		Status:    model.EnvironmentStatusRunning,
		CreatedAt: time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 18, 10, 5, 0, 0, time.UTC),
	}); err != nil {
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
	if got[0].ID != "env-new" || got[1].ID != "env-old" {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestFileStoreListReturnsEmptySliceWhenRootMissing(t *testing.T) {
	store := environment.NewFileStore(filepath.Join(t.TempDir(), "missing"))

	got, err := store.List()
	if err != nil {
		t.Fatalf("list missing root: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %d items", len(got))
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
