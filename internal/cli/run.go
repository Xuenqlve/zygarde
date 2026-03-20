package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/xuenqlve/zygarde/internal/app"
	"github.com/xuenqlve/zygarde/internal/coordinator"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

const defaultBlueprintFile = "zygarde.yaml"

// Run executes the CLI using the provided arguments.
func Run(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("expected command")
	}

	switch args[0] {
	case "blueprint":
		return runBlueprint(ctx, args[1:], stdout)
	case "template":
		return runTemplate(ctx, args[1:], stdout)
	case "create":
		return runCreate(ctx, args[1:], stdout)
	case "up":
		return runUp(ctx, args[1:], stdout)
	case "list":
		return runList(ctx, stdout)
	case "status":
		return runStatus(ctx, args[1:], stdout)
	case "doctor":
		return runDoctor(ctx, args[1:], stdout)
	case "start":
		return runStart(ctx, args[1:], stdout)
	case "stop":
		return runStop(ctx, args[1:], stdout)
	case "down":
		return runDown(ctx, args[1:], stdout)
	case "destroy":
		return runDestroy(ctx, args[1:], stdout)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runTemplate(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("expected template subcommand")
	}

	switch args[0] {
	case "list":
		return runTemplateList(ctx, args[1:], stdout)
	case "show":
		return runTemplateShow(ctx, args[1:], stdout)
	default:
		return fmt.Errorf("unknown template subcommand: %s", args[0])
	}
}

func runBlueprint(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("expected blueprint subcommand")
	}

	switch args[0] {
	case "list":
		return runBlueprintList(ctx, args[1:], stdout)
	case "show":
		return runBlueprintShow(ctx, args[1:], stdout)
	case "validate":
		return runBlueprintValidate(ctx, args[1:], stdout)
	default:
		return fmt.Errorf("unknown blueprint subcommand: %s", args[0])
	}
}

func runCreate(ctx context.Context, args []string, stdout io.Writer) error {
	return runBlueprintAction(ctx, "create", args, stdout, func(application *app.App, blueprintFile string, envType runtime.EnvironmentType) (*appResult, error) {
		result, err := application.Create(ctx, blueprintFile, envType)
		if err != nil {
			return nil, err
		}
		return &appResult{Message: result.Message}, nil
	})
}

func runUp(ctx context.Context, args []string, stdout io.Writer) error {
	return runBlueprintAction(ctx, "up", args, stdout, func(application *app.App, blueprintFile string, envType runtime.EnvironmentType) (*appResult, error) {
		result, err := application.Up(ctx, blueprintFile, envType)
		if err != nil {
			return nil, err
		}
		return &appResult{Message: result.Message}, nil
	})
}

type blueprintAction func(application *app.App, blueprintFile string, envType runtime.EnvironmentType) (*appResult, error)

func runBlueprintAction(ctx context.Context, name string, args []string, stdout io.Writer, action blueprintAction) error {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
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

	result, err := action(application, blueprintFile, runtime.EnvironmentType(envType))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout, result.Message)
	return err
}

func runBlueprintList(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("blueprint list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var root string
	fs.StringVar(&root, "dir", ".", "directory to scan for blueprint files")
	if err := fs.Parse(args); err != nil {
		return err
	}

	application, err := app.New()
	if err != nil {
		return err
	}

	result, err := application.ListBlueprints(ctx, root)
	if err != nil {
		return err
	}
	return writeBlueprintListResult(stdout, result)
}

func runBlueprintShow(ctx context.Context, args []string, stdout io.Writer) error {
	return runBlueprintInspect(ctx, "blueprint show", args, stdout, func(application *app.App, blueprintFile string, envType runtime.EnvironmentType) error {
		result, err := application.ShowBlueprint(ctx, blueprintFile, envType)
		if err != nil {
			return err
		}
		return writeBlueprintShowResult(stdout, result)
	})
}

func runBlueprintValidate(ctx context.Context, args []string, stdout io.Writer) error {
	return runBlueprintInspect(ctx, "blueprint validate", args, stdout, func(application *app.App, blueprintFile string, envType runtime.EnvironmentType) error {
		result, err := application.ValidateBlueprint(ctx, blueprintFile, envType)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, result.Message)
		return err
	})
}

type blueprintInspectAction func(application *app.App, blueprintFile string, envType runtime.EnvironmentType) error

func runBlueprintInspect(ctx context.Context, name string, args []string, stdout io.Writer, action blueprintInspectAction) error {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var blueprintFile string
	var envType string
	fs.StringVar(&blueprintFile, "f", "", "path to blueprint.yaml")
	fs.StringVar(&blueprintFile, "file", "", "path to blueprint.yaml")
	fs.StringVar(&envType, "env-type", string(runtime.EnvironmentTypeCompose), "runtime environment type")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if blueprintFile == "" && fs.NArg() > 0 {
		blueprintFile = fs.Arg(0)
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
	return action(application, blueprintFile, runtime.EnvironmentType(envType))
}

func runTemplateList(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("template list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var envType string
	fs.StringVar(&envType, "env-type", string(runtime.EnvironmentTypeCompose), "runtime environment type")
	if err := fs.Parse(args); err != nil {
		return err
	}

	application, err := app.New()
	if err != nil {
		return err
	}

	result, err := application.ListTemplates(ctx, runtime.EnvironmentType(envType))
	if err != nil {
		return err
	}
	return writeTemplateListResult(stdout, result)
}

func runTemplateShow(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("template show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var envType string
	fs.StringVar(&envType, "env-type", string(runtime.EnvironmentTypeCompose), "runtime environment type")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var middleware string
	var templateName string
	switch fs.NArg() {
	case 1:
		var err error
		middleware, templateName, err = coordinator.SplitTemplateReference(fs.Arg(0))
		if err != nil {
			return err
		}
	case 2:
		middleware = fs.Arg(0)
		templateName = fs.Arg(1)
	default:
		return fmt.Errorf("template show requires <middleware>/<template> or <middleware> <template>")
	}

	application, err := app.New()
	if err != nil {
		return err
	}

	result, err := application.ShowTemplate(ctx, middleware, templateName, runtime.EnvironmentType(envType))
	if err != nil {
		return err
	}
	return writeTemplateShowResult(stdout, result)
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

func runList(ctx context.Context, stdout io.Writer) error {
	application, err := app.New()
	if err != nil {
		return err
	}

	result, err := application.List(ctx)
	if err != nil {
		return err
	}

	return writeListResult(stdout, result)
}

func runDoctor(ctx context.Context, args []string, stdout io.Writer) error {
	return runEnvironmentAction(ctx, "doctor", args, stdout, func(application *app.App, environmentID string) (*appResult, error) {
		result, err := application.Doctor(ctx, environmentID)
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

func runDown(ctx context.Context, args []string, stdout io.Writer) error {
	return runEnvironmentAction(ctx, "down", args, stdout, func(application *app.App, environmentID string) (*appResult, error) {
		result, err := application.Down(ctx, environmentID)
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

func writeListResult(stdout io.Writer, result *coordinator.ListResult) error {
	if result == nil || len(result.Items) == 0 {
		_, err := fmt.Fprintln(stdout, "no environments found")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "ID\tNAME\tSTATUS\tRUNTIME\tBLUEPRINT\tUPDATED\tENDPOINTS"); err != nil {
		return err
	}
	for _, item := range result.Items {
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			item.Name,
			item.Status,
			item.RuntimeType,
			item.BlueprintName,
			formatListTime(item.UpdatedAt, item.CreatedAt),
			formatEndpoints(item.Endpoints),
		); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func writeBlueprintListResult(stdout io.Writer, result *coordinator.BlueprintListResult) error {
	if result == nil || len(result.Items) == 0 {
		_, err := fmt.Fprintln(stdout, "no blueprints found")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "PATH\tNAME\tVERSION\tSERVICES\tPROJECT\tDESCRIPTION"); err != nil {
		return err
	}
	for _, item := range result.Items {
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%d\t%s\t%s\n",
			item.Path,
			fallbackString(item.Name, "-"),
			fallbackString(item.Version, "-"),
			item.ServiceCount,
			fallbackString(item.ProjectName, "-"),
			fallbackString(item.Description, "-"),
		); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func writeBlueprintShowResult(stdout io.Writer, result *coordinator.BlueprintShowResult) error {
	if result == nil {
		_, err := fmt.Fprintln(stdout, "blueprint not found")
		return err
	}

	if _, err := fmt.Fprintf(stdout, "Path: %s\n", result.Path); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Name: %s\n", fallbackString(result.Name, "-")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Version: %s\n", fallbackString(result.Version, "-")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Description: %s\n", fallbackString(result.Description, "-")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Project: %s\n", fallbackString(result.ProjectName, "-")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "AutoRemove: %t\n", result.AutoRemove); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Services: %d\n", result.ServiceCount); err != nil {
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "SERVICE\tMIDDLEWARE\tTEMPLATE"); err != nil {
		return err
	}
	for _, service := range result.Services {
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\n", service.Name, service.Middleware, service.Template); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func writeTemplateListResult(stdout io.Writer, result *coordinator.TemplateListResult) error {
	if result == nil || len(result.Items) == 0 {
		_, err := fmt.Fprintln(stdout, "no templates found")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "MIDDLEWARE\tTEMPLATE\tRUNTIME\tDEFAULT\tVERSIONS\tDOC\tDESCRIPTION"); err != nil {
		return err
	}
	for _, item := range result.Items {
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%t\t%s\t%s\t%s\n",
			item.Middleware,
			item.Template,
			item.RuntimeType,
			item.Default,
			strings.Join(item.Versions, ","),
			fallbackString(item.DocPath, "-"),
			fallbackString(item.Description, "-"),
		); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func writeTemplateShowResult(stdout io.Writer, result *coordinator.TemplateShowResult) error {
	if result == nil {
		_, err := fmt.Fprintln(stdout, "template not found")
		return err
	}

	if _, err := fmt.Fprintf(stdout, "Middleware: %s\n", fallbackString(result.Middleware, "-")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Template: %s\n", fallbackString(result.Template, "-")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Runtime: %s\n", fallbackString(result.RuntimeType, "-")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Default: %t\n", result.Default); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Versions: %s\n", fallbackString(strings.Join(result.Versions, ", "), "-")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Doc: %s\n", fallbackString(result.DocPath, "-")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Description: %s\n", fallbackString(result.Description, "-")); err != nil {
		return err
	}
	return nil
}

func formatListTime(updatedAt time.Time, createdAt time.Time) string {
	ts := updatedAt
	if ts.IsZero() {
		ts = createdAt
	}
	if ts.IsZero() {
		return "-"
	}
	return ts.Format(time.RFC3339)
}

func formatEndpoints(endpoints []model.Endpoint) string {
	if len(endpoints) == 0 {
		return "-"
	}

	parts := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		label := fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
		if endpoint.Protocol != "" {
			label += "/" + endpoint.Protocol
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, ",")
}

func fallbackString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
