package compose

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

const dockerCommand = "docker"

// Executor executes Docker Compose lifecycle commands.
type Executor struct {
	runner CommandRunner
}

// NewExecutor creates a compose deployment executor.
func NewExecutor(runner CommandRunner) Executor {
	if runner == nil {
		runner = execRunner{}
	}
	return Executor{runner: runner}
}

// Apply executes `docker compose up -d`.
func (e Executor) Apply(ctx context.Context, env model.Environment, rendered model.RenderResult) (*runtime.OperationResult, error) {
	output, err := e.runCompose(ctx, env, "up", "-d")
	if err != nil {
		return nil, fmt.Errorf("compose apply: %w: %s", err, strings.TrimSpace(output))
	}

	status, statusMessage, endpoints, statusErr := e.inspectProject(ctx, env)
	if statusErr != nil {
		return nil, statusErr
	}

	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf(
			"compose apply completed for %s using %s (%s: %s)",
			env.Name,
			rendered.PrimaryFile,
			status,
			statusMessage,
		)),
		Changed:   true,
		Endpoints: endpoints,
	}, nil
}

// Status queries `docker compose ps --format json`.
func (e Executor) Status(ctx context.Context, env model.Environment) (*runtime.StatusResult, error) {
	status, message, endpoints, err := e.inspectProject(ctx, env)
	if err != nil {
		return nil, err
	}
	return &runtime.StatusResult{
		Status:    status,
		Message:   message,
		Endpoints: endpoints,
	}, nil
}

// Start executes `docker compose start`.
func (e Executor) Start(ctx context.Context, env model.Environment) (*runtime.OperationResult, error) {
	output, err := e.runCompose(ctx, env, "start")
	if err != nil {
		return nil, fmt.Errorf("compose start: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose start completed for %s: %s", env.Name, strings.TrimSpace(output))),
		Changed: true,
	}, nil
}

// Stop executes `docker compose stop`.
func (e Executor) Stop(ctx context.Context, env model.Environment) (*runtime.OperationResult, error) {
	output, err := e.runCompose(ctx, env, "stop")
	if err != nil {
		return nil, fmt.Errorf("compose stop: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose stop completed for %s: %s", env.Name, strings.TrimSpace(output))),
		Changed: true,
	}, nil
}

// Destroy executes `docker compose down`.
func (e Executor) Destroy(ctx context.Context, env model.Environment) (*runtime.OperationResult, error) {
	output, err := e.runCompose(ctx, env, "down")
	if err != nil {
		return nil, fmt.Errorf("compose destroy: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose destroy completed for %s: %s", env.Name, strings.TrimSpace(output))),
		Changed: true,
	}, nil
}

// Cleanup removes the local runtime workspace after runtime resources are destroyed.
func (e Executor) Cleanup(_ context.Context, env model.Environment) (*runtime.OperationResult, error) {
	if env.WorkspaceDir == "" {
		return nil, fmt.Errorf("compose cleanup: workspace dir is required")
	}
	if err := os.RemoveAll(env.WorkspaceDir); err != nil {
		return nil, fmt.Errorf("compose cleanup: %w", err)
	}
	return &runtime.OperationResult{
		Message: fmt.Sprintf("compose cleanup completed for %s", env.Name),
		Changed: true,
	}, nil
}

type psEntry struct {
	Name       string        `json:"Name"`
	Service    string        `json:"Service"`
	State      string        `json:"State"`
	Status     string        `json:"Status"`
	Publishers []psPublisher `json:"Publishers"`
}

type psPublisher struct {
	URL           string `json:"URL"`
	TargetPort    int    `json:"TargetPort"`
	PublishedPort int    `json:"PublishedPort"`
	Protocol      string `json:"Protocol"`
}

func (e Executor) inspectProject(ctx context.Context, env model.Environment) (model.EnvironmentStatus, string, []model.Endpoint, error) {
	output, err := e.runCompose(ctx, env, "ps", "-a", "--format", "json")
	if err != nil {
		return model.EnvironmentStatusError, "", nil, fmt.Errorf("compose status: %w: %s", err, strings.TrimSpace(output))
	}

	entries, err := parsePSOutput(output)
	if err != nil {
		return model.EnvironmentStatusError, "", nil, err
	}

	status := mapEnvironmentStatus(entries)
	message := fmt.Sprintf("compose status for %s: %s (%d service(s))", env.Name, status, len(entries))
	return status, message, buildEndpoints(entries), nil
}

func (e Executor) runCompose(ctx context.Context, env model.Environment, args ...string) (string, error) {
	baseArgs, err := composeBaseArgs(env)
	if err != nil {
		return "", err
	}
	workdir, err := filepath.Abs(env.WorkspaceDir)
	if err != nil {
		return "", fmt.Errorf("resolve compose workdir: %w", err)
	}
	return e.runner.Run(ctx, workdir, dockerCommand, append(baseArgs, args...)...)
}

func composeBaseArgs(env model.Environment) ([]string, error) {
	if env.ProjectName == "" {
		return nil, fmt.Errorf("compose project name is required")
	}
	if env.ComposeFile == "" {
		return nil, fmt.Errorf("compose file is required")
	}
	composeFile, err := filepath.Abs(env.ComposeFile)
	if err != nil {
		return nil, fmt.Errorf("resolve compose file: %w", err)
	}
	return []string{"compose", "-p", env.ProjectName, "-f", composeFile}, nil
}

func parsePSOutput(output string) ([]psEntry, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil, nil
	}

	var entries []psEntry
	if err := json.Unmarshal([]byte(trimmed), &entries); err != nil {
		var entry psEntry
		if singleErr := json.Unmarshal([]byte(trimmed), &entry); singleErr == nil {
			return []psEntry{entry}, nil
		}

		scanner := bufio.NewScanner(strings.NewReader(trimmed))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var lineEntry psEntry
			if lineErr := json.Unmarshal([]byte(line), &lineEntry); lineErr != nil {
				return nil, fmt.Errorf("parse compose ps output: %w", err)
			}
			entries = append(entries, lineEntry)
		}
		if scanErr := scanner.Err(); scanErr != nil {
			return nil, fmt.Errorf("parse compose ps output: %w", scanErr)
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("parse compose ps output: %w", err)
		}
	}
	return entries, nil
}

func mapEnvironmentStatus(entries []psEntry) model.EnvironmentStatus {
	if len(entries) == 0 {
		return model.EnvironmentStatusDestroyed
	}

	allRunning := true
	allStopped := true
	for _, entry := range entries {
		state := strings.ToLower(entry.State)
		switch state {
		case "running":
			allStopped = false
		case "exited", "stopped", "created":
			allRunning = false
		default:
			return model.EnvironmentStatusError
		}
	}

	if allRunning {
		return model.EnvironmentStatusRunning
	}
	if allStopped {
		return model.EnvironmentStatusStopped
	}
	return model.EnvironmentStatusError
}

func buildEndpoints(entries []psEntry) []model.Endpoint {
	endpoints := make([]model.Endpoint, 0)
	for _, entry := range entries {
		name := entry.Service
		if name == "" {
			name = entry.Name
		}
		for _, publisher := range entry.Publishers {
			if publisher.PublishedPort == 0 {
				continue
			}
			host := publisher.URL
			if host == "" {
				host = "127.0.0.1"
			}
			endpoints = append(endpoints, model.Endpoint{
				Name:     name,
				Host:     host,
				Port:     publisher.PublishedPort,
				Protocol: strings.ToLower(publisher.Protocol),
			})
		}
	}
	return endpoints
}
