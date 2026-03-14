package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBlueprintFileUsesDefaultFileInCurrentDirectory(t *testing.T) {
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

	path := filepath.Join(dir, defaultBlueprintFile)
	if err := os.WriteFile(path, []byte("name: demo\nservices: []\n"), 0o644); err != nil {
		t.Fatalf("write blueprint file: %v", err)
	}

	got, err := resolveBlueprintFile()
	if err != nil {
		t.Fatalf("resolve blueprint file: %v", err)
	}
	if got != defaultBlueprintFile {
		t.Fatalf("expected %q, got %q", defaultBlueprintFile, got)
	}
}

func TestResolveBlueprintFileReturnsHelpfulErrorWhenDefaultMissing(t *testing.T) {
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

	_, err = resolveBlueprintFile()
	if err == nil {
		t.Fatal("expected error when default blueprint file is missing")
	}

	want := "use -f/--file or create " + defaultBlueprintFile + " in the current directory"
	if err.Error() != "blueprint file is required: "+want {
		t.Fatalf("unexpected error: %v", err)
	}
}
