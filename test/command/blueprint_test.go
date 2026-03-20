package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuenqlve/zygarde/internal/cli"
)

func TestRunBlueprintListPrintsDiscoveredBlueprints(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	root := filepath.Join(dir, "blueprints")
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("create blueprint root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "zygarde.yaml"), []byte("name: root-demo\nversion: v1\nservices: []\n"), 0o644); err != nil {
		t.Fatalf("write root blueprint: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "cache.blueprint.yaml"), []byte("name: cache-demo\nversion: v2\nservices: []\n"), 0o644); err != nil {
		t.Fatalf("write nested blueprint: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored.yaml"), []byte("name: ignored\n"), 0o644); err != nil {
		t.Fatalf("write ignored yaml: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{"blueprint", "list", "--dir", root}, &stdout); err != nil {
		t.Fatalf("run blueprint list: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{"PATH", "NAME", "root-demo", "cache-demo", "nested"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
	if strings.Contains(output, "ignored") {
		t.Fatalf("did not expect ignored yaml to appear in blueprint list: %q", output)
	}
}

func TestRunBlueprintShowUsesDefaultBlueprintFile(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	if err := os.WriteFile(filepath.Join(dir, "zygarde.yaml"), []byte(
		"name: demo\nversion: v1\ndescription: demo blueprint\nruntime:\n  project-name: demo-project\nservices:\n  - middleware: mysql\n    template: single\n",
	), 0o644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{"blueprint", "show"}, &stdout); err != nil {
		t.Fatalf("run blueprint show: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{"Path:", "Name: demo", "Version: v1", "Project: demo-project", "SERVICE", "mysql-1", "mysql", "single"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
}

func TestRunBlueprintValidatePrintsSuccessMessage(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	if err := os.WriteFile(filepath.Join(dir, "zygarde.yaml"), []byte(
		"name: validate-demo\nversion: v1\nservices:\n  - middleware: redis\n    template: single\n",
	), 0o644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{"blueprint", "validate"}, &stdout); err != nil {
		t.Fatalf("run blueprint validate: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if !strings.Contains(output, "blueprint zygarde.yaml is valid for compose with 1 service(s)") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunBlueprintValidateReturnsHelpfulErrorForUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"blueprint", "unknown"}, &stdout)
	if err == nil {
		t.Fatal("expected blueprint subcommand error")
	}
	if err.Error() != "unknown blueprint subcommand: unknown" {
		t.Fatalf("unexpected error: %v", err)
	}
}
