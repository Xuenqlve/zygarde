package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/xuenqlve/zygarde/internal/app"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

// Run executes the CLI using the provided arguments.
func Run(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("expected command")
	}

	switch args[0] {
	case "create":
		return runCreate(ctx, args[1:], stdout)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runCreate(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var blueprintFile string
	var envType string
	fs.StringVar(&blueprintFile, "f", "", "path to blueprint.yaml")
	fs.StringVar(&blueprintFile, "file", "", "path to blueprint.yaml")
	fs.StringVar(&envType, "env-type", string(runtime.EnvironmentTypeCompose), "runtime environment type")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if blueprintFile == "" {
		return fmt.Errorf("blueprint file is required")
	}

	application, err := app.New()
	if err != nil {
		return err
	}

	result, err := application.Create(ctx, blueprintFile, runtime.EnvironmentType(envType))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "configured %d service(s) for %s\n", len(result.Blueprint.Services), result.Blueprint.Name)
	return err
}
