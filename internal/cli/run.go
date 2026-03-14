package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/xuenqlve/zygarde/internal/app"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

const defaultBlueprintFile = "zygarde.yaml"

// Run executes the CLI using the provided arguments.
func Run(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("expected command")
	}

	switch args[0] {
	case "create":
		return runCreate(ctx, args[1:], stdout)
	case "status":
		return runStatus(ctx, args[1:], stdout)
	case "start":
		return runStart(ctx, args[1:], stdout)
	case "stop":
		return runStop(ctx, args[1:], stdout)
	case "destroy":
		return runDestroy(ctx, args[1:], stdout)
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
		var err error
		blueprintFile, err = resolveBlueprintFile()
		if err != nil {
			return err
		}
	}

	application, err := app.New()
	if err != nil {
		return err
	}

	result, err := application.Create(ctx, blueprintFile, runtime.EnvironmentType(envType))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout, result.Message)
	return err
}

func runStatus(ctx context.Context, args []string, stdout io.Writer) error {
	return runEnvironmentAction(ctx, "status", args, stdout, func(application *app.App, environmentID string) (*appResult, error) {
		result, err := application.Status(ctx, environmentID)
		if err != nil {
			return nil, err
		}
		return &appResult{Message: result.Message}, nil
	})
}

func runStart(ctx context.Context, args []string, stdout io.Writer) error {
	return runEnvironmentAction(ctx, "start", args, stdout, func(application *app.App, environmentID string) (*appResult, error) {
		result, err := application.Start(ctx, environmentID)
		if err != nil {
			return nil, err
		}
		return &appResult{Message: result.Message}, nil
	})
}

func runStop(ctx context.Context, args []string, stdout io.Writer) error {
	return runEnvironmentAction(ctx, "stop", args, stdout, func(application *app.App, environmentID string) (*appResult, error) {
		result, err := application.Stop(ctx, environmentID)
		if err != nil {
			return nil, err
		}
		return &appResult{Message: result.Message}, nil
	})
}

func runDestroy(ctx context.Context, args []string, stdout io.Writer) error {
	return runEnvironmentAction(ctx, "destroy", args, stdout, func(application *app.App, environmentID string) (*appResult, error) {
		result, err := application.Destroy(ctx, environmentID)
		if err != nil {
			return nil, err
		}
		return &appResult{Message: result.Message}, nil
	})
}

func resolveBlueprintFile() (string, error) {
	info, err := os.Stat(defaultBlueprintFile)
	if err == nil {
		if info.IsDir() {
			return "", fmt.Errorf("default blueprint path is a directory: %s", defaultBlueprintFile)
		}
		return defaultBlueprintFile, nil
	}
	if os.IsNotExist(err) {
		return "", fmt.Errorf("blueprint file is required: use -f/--file or create %s in the current directory", defaultBlueprintFile)
	}
	return "", fmt.Errorf("resolve blueprint file: %w", err)
}

type appResult struct {
	Message string
}

type environmentAction func(application *app.App, environmentID string) (*appResult, error)

func runEnvironmentAction(ctx context.Context, name string, args []string, stdout io.Writer, action environmentAction) error {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var environmentID string
	fs.StringVar(&environmentID, "id", "", "environment id")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if environmentID == "" && fs.NArg() > 0 {
		environmentID = fs.Arg(0)
	}
	if environmentID == "" {
		return fmt.Errorf("environment id is required")
	}

	application, err := app.New()
	if err != nil {
		return err
	}

	result, err := action(application, environmentID)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout, result.Message)
	return err
}
