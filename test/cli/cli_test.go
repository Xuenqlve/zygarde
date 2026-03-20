package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuenqlve/zygarde/internal/cli"
)

func TestRunReturnsHelpfulErrorWhenCommandMissing(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(context.Background(), nil, &stdout)
	if err == nil {
		t.Fatal("expected error when command is missing")
	}
	if err.Error() != "expected command" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunReturnsHelpfulErrorWhenCommandUnknown(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"unknown"}, &stdout)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if err.Error() != "unknown command: unknown" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStatusUsesExplicitIDFlagWithoutCurrentEnvironment(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"status", "--id", "env-explicit"}, &stdout)
	if err == nil {
		t.Fatal("expected environment lookup error")
	}
	if strings.Contains(err.Error(), "resolve current environment") {
		t.Fatalf("expected explicit id to bypass current environment resolution, got %v", err)
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStatusUsesPositionalIDWithoutCurrentEnvironment(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"status", "env-positional"}, &stdout)
	if err == nil {
		t.Fatal("expected environment lookup error")
	}
	if strings.Contains(err.Error(), "resolve current environment") {
		t.Fatalf("expected positional id to bypass current environment resolution, got %v", err)
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStatusWithoutIDFallsBackToCurrentEnvironmentResolution(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"status"}, &stdout)
	if err == nil {
		t.Fatal("expected error when current environment marker is missing")
	}
	if !strings.Contains(err.Error(), "resolve current environment") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunUpRejectsUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"up", "--unknown"}, &stdout)
	if err == nil {
		t.Fatal("expected flag parse error")
	}
	if !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunUpWithoutBlueprintReturnsHelpfulError(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"up"}, &stdout)
	if err == nil {
		t.Fatal("expected missing blueprint error")
	}
	if !strings.Contains(err.Error(), "blueprint file is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunUpResolvesDefaultBlueprintFileBeforeAppExecution(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	path := filepath.Join(dir, "zygarde.yaml")
	if err := os.WriteFile(path, []byte("name: demo\nversion: v1\nservices: []\n"), 0o644); err != nil {
		t.Fatalf("write blueprint file: %v", err)
	}

	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"up"}, &stdout)
	if err == nil {
		t.Fatal("expected runtime create error because blueprint is incomplete for actual up")
	}
	if strings.Contains(err.Error(), "blueprint file is required") {
		t.Fatalf("expected default blueprint resolution to succeed, got %v", err)
	}
}

func TestRunCreateResolvesDefaultBlueprintFileBeforeAppExecution(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	path := filepath.Join(dir, "zygarde.yaml")
	if err := os.WriteFile(path, []byte("name: demo\nversion: v1\nservices: []\n"), 0o644); err != nil {
		t.Fatalf("write blueprint file: %v", err)
	}

	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"create"}, &stdout)
	if err == nil {
		t.Fatal("expected create reserved error")
	}
	if strings.Contains(err.Error(), "blueprint file is required") {
		t.Fatalf("expected default blueprint resolution to succeed, got %v", err)
	}
	if !strings.Contains(err.Error(), "at least one runtime context is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveBlueprintDefaultFileBehaviorThroughRun(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	path := filepath.Join(dir, "zygarde.yaml")
	if err := os.WriteFile(path, []byte("name: demo\nservices: []\n"), 0o644); err != nil {
		t.Fatalf("write blueprint file: %v", err)
	}

	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"create"}, &stdout)
	if err == nil {
		t.Fatal("expected create validation error")
	}
	if !strings.Contains(err.Error(), "at least one runtime context is required") {
		t.Fatalf("unexpected error: %v", err)
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
