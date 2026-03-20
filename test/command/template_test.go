package command

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/xuenqlve/zygarde/internal/cli"
)

func TestRunTemplateListPrintsRegisteredTemplates(t *testing.T) {
	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{"template", "list"}, &stdout); err != nil {
		t.Fatalf("run template list: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{"MIDDLEWARE", "mysql", "single", "compose", "docs/mysql.md", "redis", "cluster"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
}

func TestRunTemplateShowPrintsTemplateDetails(t *testing.T) {
	var stdout bytes.Buffer
	if err := cli.Run(context.Background(), []string{"template", "show", "mysql/single"}, &stdout); err != nil {
		t.Fatalf("run template show: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"Middleware: mysql",
		"Template: single",
		"Runtime: compose",
		"Default: true",
		"Versions: v5.7, v8.0",
		"Doc: docs/mysql.md",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
}

func TestRunTemplateShowReturnsHelpfulErrorForUnknownTemplate(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"template", "show", "mysql/unknown"}, &stdout)
	if err == nil {
		t.Fatal("expected template lookup error")
	}
	if !strings.Contains(err.Error(), "template not found: mysql/unknown/compose") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunTemplateReturnsHelpfulErrorForUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(context.Background(), []string{"template", "unknown"}, &stdout)
	if err == nil {
		t.Fatal("expected template subcommand error")
	}
	if err.Error() != "unknown template subcommand: unknown" {
		t.Fatalf("unexpected error: %v", err)
	}
}
