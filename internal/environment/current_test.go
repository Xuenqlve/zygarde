package environment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadAndClearCurrent(t *testing.T) {
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

	want := CurrentEnvironment{
		EnvironmentID: "env-1",
		WorkspaceDir:  "/tmp/env-1",
		ProjectName:   "zygarde-env-1",
	}
	if err := SaveCurrent(want); err != nil {
		t.Fatalf("save current environment: %v", err)
	}

	got, err := LoadCurrent()
	if err != nil {
		t.Fatalf("load current environment: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected current environment: got=%+v want=%+v", got, want)
	}

	if err := ClearCurrent(); err != nil {
		t.Fatalf("clear current environment: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, currentEnvironmentFile)); !os.IsNotExist(err) {
		t.Fatalf("expected current environment file removed, got err=%v", err)
	}
}
