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

// Apply executes the generated build.sh entrypoint for one compose bundle.
func (e Executor) Apply(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error) {
	env := plan.Environment
	script := plan.BuildScript
	if script == "" {
		return nil, fmt.Errorf("compose apply: build script is required")
	}
	workdir := plan.WorkspaceDir
	if workdir == "" {
		return nil, fmt.Errorf("compose apply: workspace dir is required")
	}
	scriptPath, err := filepath.Abs(script)
	if err != nil {
		return nil, fmt.Errorf("compose apply: resolve build script: %w", err)
	}
	output, err := e.runner.Run(ctx, workdir, "/bin/sh", scriptPath)
	if err != nil {
		return nil, fmt.Errorf("compose apply: %w: %s", err, strings.TrimSpace(output))
	}

	statusPlan := runtime.LifecyclePlan{
		Environment:  env,
		WorkspaceDir: plan.WorkspaceDir,
		ProjectName:  plan.ProjectName,
		PrimaryFile:  plan.PrimaryFile,
	}
	status, statusMessage, endpoints, statusErr := e.inspectProject(ctx, statusPlan)
	if statusErr != nil {
		return nil, statusErr
	}

	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf(
			"compose apply completed for %s using %s (%s: %s)",
			env.Name,
			script,
			status,
			statusMessage,
		)),
		Changed:   true,
		Endpoints: endpoints,
	}, nil
}

// Status queries `docker compose ps --format json`.
func (e Executor) Status(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.StatusResult, error) {
	status, message, endpoints, err := e.inspectProject(ctx, plan)
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
func (e Executor) Start(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	output, err := e.runCompose(ctx, plan, "start")
	if err != nil {
		return nil, fmt.Errorf("compose start: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose start completed for %s: %s", plan.Environment.Name, strings.TrimSpace(output))),
		Changed: true,
	}, nil
}

// Stop executes `docker compose stop`.
func (e Executor) Stop(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	output, err := e.runCompose(ctx, plan, "stop")
	if err != nil {
		return nil, fmt.Errorf("compose stop: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose stop completed for %s: %s", plan.Environment.Name, strings.TrimSpace(output))),
		Changed: true,
	}, nil
}

// Destroy executes `docker compose down`.
func (e Executor) Destroy(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	output, err := e.runCompose(ctx, plan, "down")
	if err != nil {
		return nil, fmt.Errorf("compose destroy: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose destroy completed for %s: %s", plan.Environment.Name, strings.TrimSpace(output))),
		Changed: true,
	}, nil
}

// Cleanup removes the local runtime workspace after runtime resources are destroyed.
func (e Executor) Cleanup(_ context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	workdir := plan.WorkspaceDir
	if workdir == "" {
		return nil, fmt.Errorf("compose cleanup: workspace dir is required")
	}
	if err := os.RemoveAll(workdir); err != nil {
		return nil, fmt.Errorf("compose cleanup: %w", err)
	}
	return &runtime.OperationResult{
		Message: fmt.Sprintf("compose cleanup completed for %s", plan.Environment.Name),
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

func (e Executor) inspectProject(ctx context.Context, plan runtime.LifecyclePlan) (model.EnvironmentStatus, string, []model.Endpoint, error) {
	output, err := e.runCompose(ctx, plan, "ps", "-a", "--format", "json")
	if err != nil {
		return model.EnvironmentStatusError, "", nil, fmt.Errorf("compose status: %w: %s", err, strings.TrimSpace(output))
	}

	entries, err := parsePSOutput(output)
	if err != nil {
		return model.EnvironmentStatusError, "", nil, err
	}

	status := mapEnvironmentStatus(entries)
	message := fmt.Sprintf("compose status for %s: %s (%d service(s))", plan.Environment.Name, status, len(entries))
	return status, message, buildEndpoints(entries), nil
}

func (e Executor) runCompose(ctx context.Context, plan runtime.LifecyclePlan, args ...string) (string, error) {
	baseArgs, err := composeBaseArgs(plan)
	if err != nil {
		return "", err
	}
	workdir := plan.WorkspaceDir
	if workdir == "" {
		return "", fmt.Errorf("compose workdir is required")
	}
	workdir, err = filepath.Abs(workdir)
	if err != nil {
		return "", fmt.Errorf("resolve compose workdir: %w", err)
	}
	return e.runner.Run(ctx, workdir, dockerCommand, append(baseArgs, args...)...)
}

func composeBaseArgs(plan runtime.LifecyclePlan) ([]string, error) {
	projectName := plan.ProjectName
	if projectName == "" {
		return nil, fmt.Errorf("compose project name is required")
	}
	if plan.PrimaryFile == "" {
		return nil, fmt.Errorf("compose file is required")
	}
	composeFile, err := filepath.Abs(plan.PrimaryFile)
	if err != nil {
		return nil, fmt.Errorf("resolve compose file: %w", err)
	}
	return []string{"compose", "-p", projectName, "-f", composeFile}, nil
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
