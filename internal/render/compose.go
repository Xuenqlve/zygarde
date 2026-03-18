package render

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xuenqlve/zygarde/internal/runtime"
	"gopkg.in/yaml.v2"
)

const defaultContainerEngine = "docker"

// ComposeRenderer generates a minimal docker-compose artifact for the create flow.
type ComposeRenderer struct {
	containerEngine string
}

// NewComposeRenderer creates a compose renderer instance.
func NewComposeRenderer(containerEngine string) ComposeRenderer {
	if containerEngine == "" {
		containerEngine = defaultContainerEngine
	}
	return ComposeRenderer{containerEngine: containerEngine}
}

// Render builds a minimal compose document from normalized runtime contexts.
func (r ComposeRenderer) Render(_ context.Context, req Request) (*runtime.RenderPlan, error) {
	services := make([]string, 0, len(req.Contexts))
	document := composeDocument{
		Services: make(map[string]composeService, len(req.Contexts)),
	}
	pool := newAssetPool()

	for _, contextItem := range req.Contexts {
		item, ok := contextItem.(runtime.ComposeContext)
		if !ok {
			return nil, fmt.Errorf("render compose: unsupported context type %T", contextItem)
		}

		renderInput := item.RenderInput()
		serviceName := renderInput.ServiceName
		if serviceName == "" {
			return nil, fmt.Errorf("render compose: service name is required")
		}
		if renderInput.Service.Image == "" {
			return nil, fmt.Errorf("render compose: service %s image is required", serviceName)
		}
		services = append(services, serviceName)
		document.Services[serviceName] = toComposeService(renderInput.Service)
		for _, asset := range renderInput.Assets {
			pool.add(asset)
		}
	}

	content, err := yaml.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("render compose: marshal document: %w", err)
	}

	assets := make([]runtime.RenderedAsset, 0, 5+len(pool.unique))
	if err := writeAsset(req.Prepared.Layout.ComposeFile, string(content), 0o644); err != nil {
		return nil, err
	}
	assets = append(assets, runtime.RenderedAsset{Path: req.Prepared.Layout.ComposeFile, Mode: 0o644})

	envContent, err := mergeEnvAssets(pool.grouped["env_file"])
	if err != nil {
		return nil, err
	}
	if err := writeAsset(req.Prepared.Layout.EnvFile, envContent, 0o644); err != nil {
		return nil, err
	}
	assets = append(assets, runtime.RenderedAsset{Path: req.Prepared.Layout.EnvFile, Mode: 0o644})

	buildContent := mergeScriptAssets(
		r.containerEngine,
		pool.grouped["build_script"],
		fmt.Sprintf("\"$CONTAINER_ENGINE\" compose -p %q -f %q up -d\n", req.Prepared.ProjectName, filepath.Base(req.Prepared.Layout.ComposeFile)),
	)
	if err := writeAsset(req.Prepared.Layout.BuildScript, buildContent, 0o755); err != nil {
		return nil, err
	}
	assets = append(assets, runtime.RenderedAsset{Path: req.Prepared.Layout.BuildScript, Mode: 0o755})

	checkContent := mergeScriptAssets(
		r.containerEngine,
		pool.grouped["check_script"],
		fmt.Sprintf("\"$CONTAINER_ENGINE\" compose -p %q -f %q ps\n", req.Prepared.ProjectName, filepath.Base(req.Prepared.Layout.ComposeFile)),
	)
	if err := writeAsset(req.Prepared.Layout.CheckScript, checkContent, 0o755); err != nil {
		return nil, err
	}
	assets = append(assets, runtime.RenderedAsset{Path: req.Prepared.Layout.CheckScript, Mode: 0o755})

	readmeContent := mergeReadmeAssets(pool.grouped["readme_file"], services)
	if err := writeAsset(req.Prepared.Layout.ReadmeFile, readmeContent, 0o644); err != nil {
		return nil, err
	}
	assets = append(assets, runtime.RenderedAsset{Path: req.Prepared.Layout.ReadmeFile, Mode: 0o644})

	for _, asset := range pool.unique {
		targetPath := filepath.Join(req.Prepared.Layout.RenderDir, asset.FileName)
		if err := writeAsset(targetPath, asset.Content, asset.Mode); err != nil {
			return nil, err
		}
		assets = append(assets, runtime.RenderedAsset{Path: targetPath, Mode: asset.Mode})
	}

	return &runtime.RenderPlan{
		Prepared:       req.Prepared,
		Content:        string(content),
		PrimaryFile:    req.Prepared.Layout.ComposeFile,
		BuildScript:    req.Prepared.Layout.BuildScript,
		CheckScript:    req.Prepared.Layout.CheckScript,
		ComposeVersion: "v2",
		Services:       services,
		Assets:         assets,
	}, nil
}

type assetPool struct {
	grouped map[string][]runtime.AssetSpec
	unique  []runtime.AssetSpec
}

func newAssetPool() assetPool {
	return assetPool{grouped: make(map[string][]runtime.AssetSpec)}
}

func (p *assetPool) add(asset runtime.AssetSpec) {
	if asset.MergeMode == runtime.AssetMergeUnique {
		p.unique = append(p.unique, asset)
		return
	}
	p.grouped[asset.PathKey] = append(p.grouped[asset.PathKey], asset)
}

type composeDocument struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Image         string            `yaml:"image,omitempty"`
	Platform      string            `yaml:"platform,omitempty"`
	Hostname      string            `yaml:"hostname,omitempty"`
	ContainerName string            `yaml:"container_name,omitempty"`
	Restart       string            `yaml:"restart,omitempty"`
	Ports         []string          `yaml:"ports,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	Command       []string          `yaml:"command,omitempty"`
	HealthCheck   *composeHealth    `yaml:"healthcheck,omitempty"`
}

type composeHealth struct {
	Test        []string `yaml:"test,omitempty"`
	Interval    string   `yaml:"interval,omitempty"`
	Timeout     string   `yaml:"timeout,omitempty"`
	Retries     int      `yaml:"retries,omitempty"`
	StartPeriod string   `yaml:"start_period,omitempty"`
}

func toComposeService(spec runtime.ServiceSpec) composeService {
	service := composeService{
		Image:         spec.Image,
		Platform:      spec.Platform,
		Hostname:      spec.Hostname,
		ContainerName: spec.ContainerName,
		Restart:       spec.Restart,
		Environment:   spec.Environment,
		Command:       spec.Command,
	}

	if len(spec.Ports) > 0 {
		service.Ports = make([]string, 0, len(spec.Ports))
		for _, port := range spec.Ports {
			binding := fmt.Sprintf("%d:%d", port.HostPort, port.ContainerPort)
			if port.Protocol != "" && port.Protocol != "tcp" {
				binding += "/" + port.Protocol
			}
			service.Ports = append(service.Ports, binding)
		}
	}

	if len(spec.Volumes) > 0 {
		service.Volumes = make([]string, 0, len(spec.Volumes))
		for _, volume := range spec.Volumes {
			mount := fmt.Sprintf("%s:%s", volume.Source, volume.Target)
			if volume.ReadOnly {
				mount += ":ro"
			}
			service.Volumes = append(service.Volumes, mount)
		}
	}

	if spec.HealthCheck != nil {
		service.HealthCheck = &composeHealth{
			Test:        spec.HealthCheck.Test,
			Interval:    spec.HealthCheck.Interval,
			Timeout:     spec.HealthCheck.Timeout,
			Retries:     spec.HealthCheck.Retries,
			StartPeriod: spec.HealthCheck.StartPeriod,
		}
	}

	return service
}

func mergeEnvAssets(items []runtime.AssetSpec) (string, error) {
	if len(items) == 0 {
		return "", nil
	}
	values := make(map[string]string)
	for _, item := range items {
		for _, line := range strings.Split(item.Content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				return "", fmt.Errorf("render compose: invalid env line %q", line)
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if existing, ok := values[key]; ok && existing != value {
				return "", fmt.Errorf("render compose: env conflict for %s", key)
			}
			values[key] = value
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func mergeScriptAssets(containerEngine string, items []runtime.AssetSpec, fallback string) string {
	if containerEngine == "" {
		containerEngine = defaultContainerEngine
	}
	var builder strings.Builder
	builder.WriteString("#!/usr/bin/env bash\n")
	builder.WriteString("set -euo pipefail\n\n")
	builder.WriteString("ROOT_DIR=\"$(cd \"$(dirname \"$0\")\" && pwd)\"\n")
	builder.WriteString("cd \"$ROOT_DIR\"\n\n")
	builder.WriteString(fmt.Sprintf("CONTAINER_ENGINE=\"${ZYGARDE_CONTAINER_ENGINE:-%s}\"\n\n", containerEngine))
	builder.WriteString("if [ -f .env ]; then\n")
	builder.WriteString("    set -a\n")
	builder.WriteString("    . ./.env\n")
	builder.WriteString("    set +a\n")
	builder.WriteString("fi\n\n")
	builder.WriteString(fallback)
	if len(items) > 0 {
		builder.WriteString("\n")
	}
	for _, item := range items {
		builder.WriteString(strings.TrimSpace(item.Content))
		builder.WriteString("\n\n")
	}
	return builder.String()
}

func mergeReadmeAssets(items []runtime.AssetSpec, services []string) string {
	if len(items) == 0 {
		return fmt.Sprintf("# Compose Stack\n\nServices: %s\n", strings.Join(services, ", "))
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		parts = append(parts, content)
	}
	return strings.Join(parts, "\n\n") + "\n"
}

func writeAsset(path, content string, mode int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("render compose: create asset dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), os.FileMode(mode)); err != nil {
		return fmt.Errorf("render compose: write asset %s: %w", path, err)
	}
	return nil
}
