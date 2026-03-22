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

func TestRunBlueprintShowResolvesBlueprintByName(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	root := filepath.Join(dir, "blueprints")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create blueprint root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "demo.blueprint.yaml"), []byte(
		"name: demo-stack\nversion: v1\nservices:\n  - middleware: redis\n    template: single\n",
	), 0o644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{"blueprint", "show", "demo-stack"}, &stdout); err != nil {
		t.Fatalf("run blueprint show by name: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{"Path: blueprints/demo.blueprint.yaml", "Name: demo-stack", "redis-1"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
}

func TestRunBlueprintCreateWritesBlueprintSkeleton(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{
		"blueprint", "create", "demo-stack",
		"--dir", "blueprints",
		"--middleware", "mysql",
		"--template", "single",
		"--version", "v8.0",
		"--description", "demo blueprint",
	}, &stdout); err != nil {
		t.Fatalf("run blueprint create: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if !strings.Contains(output, "created blueprint blueprints/demo-stack.blueprint.yaml for demo-stack") {
		t.Fatalf("unexpected output: %q", output)
	}

	data, err := os.ReadFile(filepath.Join(dir, "blueprints", "demo-stack.blueprint.yaml"))
	if err != nil {
		t.Fatalf("read created blueprint: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"name: demo-stack",
		"version: v1",
		"description: demo blueprint",
		"project-name: demo-stack",
		"middleware: mysql",
		"template: single",
		"version: v8.0",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected content to contain %q, got %q", want, content)
		}
	}
}

func TestRunBlueprintCopyCreatesDerivedBlueprint(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	root := filepath.Join(dir, "blueprints")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create blueprint root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "demo.blueprint.yaml"), []byte(
		"name: demo-stack\nversion: v1\nruntime:\n  project-name: demo-project\nservices:\n  - name: redis-1\n    middleware: redis\n    template: single\n",
	), 0o644); err != nil {
		t.Fatalf("write source blueprint: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{
		"blueprint", "copy", "demo-stack",
		"--name", "demo-stack-copy",
		"--project-name", "demo-project-copy",
	}, &stdout); err != nil {
		t.Fatalf("run blueprint copy: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if !strings.Contains(output, "copied blueprint to demo-stack-copy.blueprint.yaml for demo-stack-copy") {
		t.Fatalf("unexpected output: %q", output)
	}

	data, err := os.ReadFile(filepath.Join(dir, "demo-stack-copy.blueprint.yaml"))
	if err != nil {
		t.Fatalf("read copied blueprint: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"name: demo-stack-copy",
		"project-name: demo-project-copy",
		"name: redis-1",
		"middleware: redis",
		"template: single",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected content to contain %q, got %q", want, content)
		}
	}
}

func TestRunBlueprintEditRequiresEditorEnv(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	if err := os.WriteFile(filepath.Join(dir, "zygarde.yaml"), []byte("name: demo\nversion: v1\nservices: []\n"), 0o644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"blueprint", "edit"}, &stdout)
	if err == nil {
		t.Fatal("expected missing editor error")
	}
	if !strings.Contains(err.Error(), "requires VISUAL or EDITOR") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBlueprintEditLaunchesEditorWithResolvedPath(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	root := filepath.Join(dir, "blueprints")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create blueprint root: %v", err)
	}
	path := filepath.Join(root, "demo.blueprint.yaml")
	if err := os.WriteFile(path, []byte("name: demo-stack\nversion: v1\nservices: []\n"), 0o644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}

	captured := filepath.Join(dir, "editor-arg.txt")
	editor := filepath.Join(dir, "editor.sh")
	script := "#!/bin/sh\nprintf '%s' \"$1\" > " + shellEscapePath(captured) + "\n"
	if err := os.WriteFile(editor, []byte(script), 0o755); err != nil {
		t.Fatalf("write editor script: %v", err)
	}
	t.Setenv("VISUAL", editor)
	t.Setenv("EDITOR", "")

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{"blueprint", "edit", "demo-stack"}, &stdout); err != nil {
		t.Fatalf("run blueprint edit: %v", err)
	}
	if !strings.Contains(stdout.String(), "opened blueprint blueprints/demo.blueprint.yaml in editor") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}

	data, err := os.ReadFile(captured)
	if err != nil {
		t.Fatalf("read captured editor path: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != "blueprints/demo.blueprint.yaml" {
		t.Fatalf("unexpected editor arg: %q", got)
	}
}

func TestRunBlueprintDeleteRemovesBlueprintByName(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	root := filepath.Join(dir, "blueprints")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create blueprint root: %v", err)
	}
	path := filepath.Join(root, "demo.blueprint.yaml")
	if err := os.WriteFile(path, []byte("name: demo-stack\nversion: v1\nservices: []\n"), 0o644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{"blueprint", "delete", "demo-stack"}, &stdout); err != nil {
		t.Fatalf("run blueprint delete: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected blueprint to be removed, stat err=%v", err)
	}
	if !strings.Contains(stdout.String(), "deleted blueprint blueprints/demo.blueprint.yaml for demo-stack") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestRunBlueprintUpdateRewritesBlueprintByName(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	root := filepath.Join(dir, "blueprints")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create blueprint root: %v", err)
	}
	path := filepath.Join(root, "demo.blueprint.yaml")
	if err := os.WriteFile(path, []byte(
		"name: demo-stack\nversion: v1\ndescription: old\nruntime:\n  project-name: old-project\nservices: []\n",
	), 0o644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{
		"blueprint", "update", "demo-stack",
		"--description", "new description",
		"--project-name", "new-project",
		"--name", "demo-stack-v2",
	}, &stdout); err != nil {
		t.Fatalf("run blueprint update: %v", err)
	}
	if !strings.Contains(stdout.String(), "updated blueprint blueprints/demo.blueprint.yaml for demo-stack-v2") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated blueprint: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"name: demo-stack-v2",
		"description: new description",
		"project-name: new-project",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected updated content to contain %q, got %q", want, content)
		}
	}
}

func TestRunBlueprintUpdateAddsService(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	path := filepath.Join(dir, "demo.blueprint.yaml")
	if err := os.WriteFile(path, []byte("name: demo-stack\nversion: v1\nservices: []\n"), 0o644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{
		"blueprint", "update", "demo-stack",
		"--add-service", "mysql-1",
		"--middleware", "mysql",
		"--template", "single",
		"--set", "version=v8.0",
		"--set", "port=3306",
	}, &stdout); err != nil {
		t.Fatalf("run blueprint update add-service: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated blueprint: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"name: mysql-1",
		"middleware: mysql",
		"template: single",
		"version: v8.0",
		"port: \"3306\"",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected content to contain %q, got %q", want, content)
		}
	}
}

func TestRunBlueprintUpdateModifiesExistingServiceValues(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	path := filepath.Join(dir, "demo.blueprint.yaml")
	if err := os.WriteFile(path, []byte(
		"name: demo-stack\nversion: v1\nservices:\n  - name: redis-1\n    middleware: redis\n    template: single\n    values:\n      version: v6.2\n",
	), 0o644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{
		"blueprint", "update", "demo-stack",
		"--service", "redis-1",
		"--template", "cluster",
		"--set", "version=v7.4",
	}, &stdout); err != nil {
		t.Fatalf("run blueprint update service: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated blueprint: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"name: redis-1",
		"template: cluster",
		"version: v7.4",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected content to contain %q, got %q", want, content)
		}
	}
}

func TestRunBlueprintUpdateRemovesService(t *testing.T) {
	dir := t.TempDir()
	restore := chdirForTest(t, dir)
	defer restore()

	path := filepath.Join(dir, "demo.blueprint.yaml")
	if err := os.WriteFile(path, []byte(
		"name: demo-stack\nversion: v1\nservices:\n  - name: mysql-1\n    middleware: mysql\n    template: single\n  - name: redis-1\n    middleware: redis\n    template: single\n",
	), 0o644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}

	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{
		"blueprint", "update", "demo-stack",
		"--remove-service", "mysql-1",
	}, &stdout); err != nil {
		t.Fatalf("run blueprint update remove-service: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated blueprint: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "name: mysql-1") {
		t.Fatalf("expected mysql-1 to be removed, got %q", content)
	}
	if !strings.Contains(content, "name: redis-1") {
		t.Fatalf("expected redis-1 to remain, got %q", content)
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

func shellEscapePath(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}
