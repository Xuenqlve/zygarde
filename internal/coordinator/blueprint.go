package coordinator

import (
	"context"
	"strconv"

	"github.com/xuenqlve/zygarde/internal/blueprint"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	"github.com/xuenqlve/zygarde/internal/template"
)

// BlueprintListItem captures one blueprint summary discovered from local storage.
type BlueprintListItem struct {
	Path         string
	Name         string
	Version      string
	Description  string
	ProjectName  string
	ServiceCount int
}

// BlueprintListResult contains the discovered blueprint summaries.
type BlueprintListResult struct {
	Items []BlueprintListItem
}

// BlueprintServiceItem captures one normalized service entry for show output.
type BlueprintServiceItem struct {
	Name       string
	Middleware string
	Template   string
}

// BlueprintShowResult captures one blueprint summary plus normalized services.
type BlueprintShowResult struct {
	Path         string
	Name         string
	Version      string
	Description  string
	ProjectName  string
	AutoRemove   bool
	ServiceCount int
	Services     []BlueprintServiceItem
}

// BlueprintValidateResult captures one validate action result.
type BlueprintValidateResult struct {
	Path         string
	Name         string
	RuntimeType  string
	ServiceCount int
	Message      string
}

// ListBlueprints lists blueprint files from one local root.
func (c Coordinator) ListBlueprints(_ context.Context, root string) (*BlueprintListResult, error) {
	files, err := c.blueprints.ListBlueprints(root)
	if err != nil {
		return nil, err
	}

	items := make([]BlueprintListItem, 0, len(files))
	for _, file := range files {
		items = append(items, BlueprintListItem{
			Path:         file.Path,
			Name:         file.Blueprint.Name,
			Version:      file.Blueprint.Version,
			Description:  file.Blueprint.Description,
			ProjectName:  file.Blueprint.Runtime.ProjectName,
			ServiceCount: len(file.Blueprint.Services),
		})
	}
	return &BlueprintListResult{Items: items}, nil
}

// ShowBlueprint loads one blueprint and returns its normalized summary.
func (c Coordinator) ShowBlueprint(_ context.Context, path string, envType runtime.EnvironmentType) (*BlueprintShowResult, error) {
	loaded, err := c.blueprints.LoadBlueprint(path)
	if err != nil {
		return nil, err
	}

	normalized, err := blueprint.Normalize(loaded, envType)
	if err != nil {
		return nil, err
	}

	services := make([]BlueprintServiceItem, 0, len(normalized.Services))
	for _, service := range normalized.Services {
		services = append(services, BlueprintServiceItem{
			Name:       service.Name,
			Middleware: service.Middleware,
			Template:   service.Template,
		})
	}

	return &BlueprintShowResult{
		Path:         path,
		Name:         normalized.Name,
		Version:      normalized.Version,
		Description:  normalized.Description,
		ProjectName:  normalized.Runtime.ProjectName,
		AutoRemove:   normalized.Runtime.AutoRemove,
		ServiceCount: len(normalized.Services),
		Services:     services,
	}, nil
}

// ValidateBlueprint validates one blueprint structurally for the target runtime.
func (c Coordinator) ValidateBlueprint(_ context.Context, path string, envType runtime.EnvironmentType) (*BlueprintValidateResult, error) {
	loaded, err := c.blueprints.LoadBlueprint(path)
	if err != nil {
		return nil, err
	}

	normalized, err := blueprint.Normalize(loaded, envType)
	if err != nil {
		return nil, err
	}

	for _, service := range normalized.Services {
		if _, err := template.GetMiddleware(
			template.NewMiddlewareRuntimeKey(service.Middleware, service.Template, envType),
		); err != nil {
			return nil, err
		}
	}
	if _, err := c.runtimes.Get(envType); err != nil {
		return nil, err
	}

	return &BlueprintValidateResult{
		Path:         path,
		Name:         normalized.Name,
		RuntimeType:  string(envType),
		ServiceCount: len(normalized.Services),
		Message:      buildBlueprintValidateMessage(path, normalized, envType),
	}, nil
}

func buildBlueprintValidateMessage(path string, blueprint model.Blueprint, envType runtime.EnvironmentType) string {
	return "blueprint " + path + " is valid for " + string(envType) + " with " + strconv.Itoa(len(blueprint.Services)) + " service(s)"
}
