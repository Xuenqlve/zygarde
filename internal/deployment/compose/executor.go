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

const defaultContainerEngine = "docker"

type composeEngineExecutor interface {
	Create(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error)
	Apply(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error)
	Status(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.StatusResult, error)
	Doctor(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error)
	Start(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error)
	Stop(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error)
	Destroy(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error)
	Cleanup(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error)
}

// Executor dispatches compose lifecycle actions to the engine-specific implementation.
type Executor struct {
	impl composeEngineExecutor
}

// NewExecutor creates a compose deployment executor.
func NewExecutor(containerEngine string, runner CommandRunner) Executor {
	engine := strings.ToLower(strings.TrimSpace(containerEngine))
	if engine == "" {
		engine = defaultContainerEngine
	}
	if runner == nil {
		runner = execRunner{}
	}
	switch engine {
	case "podman":
		return Executor{impl: newPodmanExecutor(runner)}
	default:
		return Executor{impl: newDockerExecutor(engine, runner)}
	}
}

func (e Executor) Create(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error) {
	return e.impl.Create(ctx, plan)
}

func (e Executor) Apply(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error) {
	return e.impl.Apply(ctx, plan)
}

func (e Executor) Status(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.StatusResult, error) {
	return e.impl.Status(ctx, plan)
}

func (e Executor) Doctor(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.impl.Doctor(ctx, plan)
}

func (e Executor) Start(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.impl.Start(ctx, plan)
}

func (e Executor) Stop(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.impl.Stop(ctx, plan)
}

func (e Executor) Destroy(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.impl.Destroy(ctx, plan)
}

func (e Executor) Cleanup(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.impl.Cleanup(ctx, plan)
}

type composeExecutorBase struct {
	runner          CommandRunner
	containerEngine string
}

type dockerExecutor struct {
	base composeExecutorBase
}

type podmanExecutor struct {
	base composeExecutorBase
}

func newDockerExecutor(containerEngine string, runner CommandRunner) composeEngineExecutor {
	return dockerExecutor{
		base: composeExecutorBase{
			runner:          runner,
			containerEngine: containerEngine,
		},
	}
}

func newPodmanExecutor(runner CommandRunner) composeEngineExecutor {
	return podmanExecutor{
		base: composeExecutorBase{
			runner:          runner,
			containerEngine: "podman",
		},
	}
}

func (e dockerExecutor) Create(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error) {
	return e.base.create(ctx, plan)
}

func (e dockerExecutor) Apply(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error) {
	return e.base.apply(ctx, plan)
}

func (e dockerExecutor) Status(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.StatusResult, error) {
	return e.base.status(ctx, plan)
}

func (e dockerExecutor) Doctor(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.base.doctor(ctx, plan)
}

func (e dockerExecutor) Start(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.base.startWithArgs(ctx, plan, "start")
}

func (e dockerExecutor) Stop(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.base.stop(ctx, plan)
}

func (e dockerExecutor) Destroy(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.base.destroy(ctx, plan)
}

func (e dockerExecutor) Cleanup(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.base.cleanup(ctx, plan)
}

func (e podmanExecutor) Create(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error) {
	return e.base.create(ctx, plan)
}

func (e podmanExecutor) Apply(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error) {
	return e.base.apply(ctx, plan)
}

func (e podmanExecutor) Status(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.StatusResult, error) {
	return e.base.status(ctx, plan)
}

func (e podmanExecutor) Doctor(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.base.doctor(ctx, plan)
}

// Podman start uses `compose up -d` because external compose providers do not
// consistently preserve `compose create -> compose start` semantics.
func (e podmanExecutor) Start(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.base.startWithArgs(ctx, plan, "up", "-d")
}

func (e podmanExecutor) Stop(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.base.stop(ctx, plan)
}

// Podman external compose providers can report a missing network after the
// containers are already stopped/removed. Treat that as an idempotent success.
func (e podmanExecutor) Destroy(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	output, err := e.base.runCompose(ctx, plan, "down")
	if err != nil {
		if isIgnorablePodmanDestroyError(output) {
			return &runtime.OperationResult{
				Message: strings.TrimSpace(fmt.Sprintf("compose destroy completed for %s: %s", plan.Environment.Name, strings.TrimSpace(output))),
				Changed: true,
			}, nil
		}
		return nil, fmt.Errorf("compose destroy: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose destroy completed for %s: %s", plan.Environment.Name, strings.TrimSpace(output))),
		Changed: true,
	}, nil
}

func (e podmanExecutor) Cleanup(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	return e.base.cleanup(ctx, plan)
}

func (e composeExecutorBase) apply(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error) {
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

func (e composeExecutorBase) create(ctx context.Context, plan runtime.ApplyPlan) (*runtime.OperationResult, error) {
	output, err := e.runComposeApply(ctx, plan, "create")
	if err != nil {
		return nil, fmt.Errorf("compose create: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose create completed for %s: %s", plan.Environment.Name, strings.TrimSpace(output))),
		Changed: true,
	}, nil
}

func (e composeExecutorBase) status(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.StatusResult, error) {
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

func (e composeExecutorBase) doctor(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	script, err := resolveLifecycleScript(plan.WorkspaceDir, "check.sh")
	if err != nil {
		return nil, fmt.Errorf("compose doctor: %w", err)
	}
	output, err := e.runner.Run(ctx, plan.WorkspaceDir, "/bin/sh", script)
	if err != nil {
		return nil, fmt.Errorf("compose doctor: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose doctor passed for %s: %s", plan.Environment.Name, strings.TrimSpace(output))),
		Changed: false,
	}, nil
}

func (e composeExecutorBase) startWithArgs(ctx context.Context, plan runtime.LifecyclePlan, args ...string) (*runtime.OperationResult, error) {
	output, err := e.runCompose(ctx, plan, args...)
	if err != nil {
		return nil, fmt.Errorf("compose start: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose start completed for %s: %s", plan.Environment.Name, strings.TrimSpace(output))),
		Changed: true,
	}, nil
}

func (e composeExecutorBase) stop(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	output, err := e.runCompose(ctx, plan, "stop")
	if err != nil {
		return nil, fmt.Errorf("compose stop: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose stop completed for %s: %s", plan.Environment.Name, strings.TrimSpace(output))),
		Changed: true,
	}, nil
}

func (e composeExecutorBase) destroy(ctx context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
	output, err := e.runCompose(ctx, plan, "down")
	if err != nil {
		return nil, fmt.Errorf("compose destroy: %w: %s", err, strings.TrimSpace(output))
	}
	return &runtime.OperationResult{
		Message: strings.TrimSpace(fmt.Sprintf("compose destroy completed for %s: %s", plan.Environment.Name, strings.TrimSpace(output))),
		Changed: true,
	}, nil
}

func (e composeExecutorBase) cleanup(_ context.Context, plan runtime.LifecyclePlan) (*runtime.OperationResult, error) {
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

func (e composeExecutorBase) inspectProject(ctx context.Context, plan runtime.LifecyclePlan) (model.EnvironmentStatus, string, []model.Endpoint, error) {
	output, err := e.runCompose(ctx, plan, "ps", "-a", "--format", "json")
	if err != nil {
		return model.EnvironmentStatusError, "", nil, fmt.Errorf("compose status: %w: %s", err, strings.TrimSpace(output))
	}

	entries, err := parsePSOutput(output)
	if err != nil {
		return model.EnvironmentStatusError, "", nil, err
	}
	if len(entries) == 0 {
		if _, statErr := os.Stat(plan.WorkspaceDir); statErr == nil {
			return model.EnvironmentStatusStopped, fmt.Sprintf("compose status for %s: %s (0 service(s))", plan.Environment.Name, model.EnvironmentStatusStopped), nil, nil
		}
	}

	status := mapEnvironmentStatus(entries)
	message := fmt.Sprintf("compose status for %s: %s (%d service(s))", plan.Environment.Name, status, len(entries))
	return status, message, buildEndpoints(entries), nil
}

func (e composeExecutorBase) runCompose(ctx context.Context, plan runtime.LifecyclePlan, args ...string) (string, error) {
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
	return e.runner.Run(ctx, workdir, e.containerEngine, append(baseArgs, args...)...)
}

func (e composeExecutorBase) runComposeApply(ctx context.Context, plan runtime.ApplyPlan, args ...string) (string, error) {
	baseArgs, err := composeBaseArgs(runtime.LifecyclePlan{
		Environment:  plan.Environment,
		WorkspaceDir: plan.WorkspaceDir,
		ProjectName:  plan.ProjectName,
		PrimaryFile:  plan.PrimaryFile,
	})
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
	return e.runner.Run(ctx, workdir, e.containerEngine, append(baseArgs, args...)...)
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

func isIgnorablePodmanDestroyError(output string) bool {
	lower := strings.ToLower(stripANSI(output))
	return strings.Contains(lower, "network not found") ||
		strings.Contains(lower, "no such network") ||
		strings.Contains(lower, "unable to find network with name or id")
}

func parsePSOutput(output string) ([]psEntry, error) {
	trimmed := strings.TrimSpace(normalizePSOutput(output))
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

func stripANSI(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))

	for i := 0; i < len(value); i++ {
		if value[i] != 0x1b {
			builder.WriteByte(value[i])
			continue
		}
		i++
		if i >= len(value) || value[i] != '[' {
			continue
		}
		for i+1 < len(value) {
			i++
			ch := value[i]
			if ch >= '@' && ch <= '~' {
				break
			}
		}
	}

	return builder.String()
}

func normalizePSOutput(value string) string {
	cleaned := stripANSI(value)
	lines := strings.Split(cleaned, "\n")
	filtered := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
			filtered = append(filtered, line)
		}
	}

	return strings.Join(filtered, "\n")
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

func resolveLifecycleScript(workdir, name string) (string, error) {
	if workdir == "" {
		return "", fmt.Errorf("workspace dir is required")
	}
	scriptPath := filepath.Join(workdir, name)
	info, err := os.Stat(scriptPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", scriptPath)
	}
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", name, err)
	}
	return absPath, nil
}
