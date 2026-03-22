package coordinator

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/xuenqlve/zygarde/internal/blueprint"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	"github.com/xuenqlve/zygarde/internal/store"
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

// BlueprintCreateRequest describes one blueprint file creation request.
type BlueprintCreateRequest struct {
	Name        string
	Description string
	ProjectName string
	Root        string
	Path        string
	Middleware  string
	Template    string
	Version     string
}

// BlueprintCreateResult captures one created blueprint file.
type BlueprintCreateResult struct {
	Path    string
	Name    string
	Message string
}

// BlueprintDeleteResult captures one deleted blueprint file.
type BlueprintDeleteResult struct {
	Path    string
	Name    string
	Message string
}

// BlueprintUpdateRequest describes one structured blueprint update.
type BlueprintUpdateRequest struct {
	Reference         string
	Root              string
	Name              string
	Description       string
	ProjectName       string
	ServiceName       string
	AddServiceName    string
	RemoveServiceName string
	Middleware        string
	Template          string
	SetValues         map[string]string
}

// BlueprintUpdateResult captures one updated blueprint file.
type BlueprintUpdateResult struct {
	Path    string
	Name    string
	Message string
}

// BlueprintCopyRequest describes one blueprint copy request.
type BlueprintCopyRequest struct {
	Reference   string
	Root        string
	Name        string
	Path        string
	ProjectName string
}

// BlueprintCopyResult captures one copied blueprint file.
type BlueprintCopyResult struct {
	Path    string
	Name    string
	Message string
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

// ResolveBlueprint resolves one blueprint reference by path or name.
func (c Coordinator) ResolveBlueprint(_ context.Context, ref string, root string) (store.BlueprintFile, error) {
	return c.blueprints.ResolveBlueprint(ref, root)
}

// CreateBlueprint creates one new local blueprint file skeleton.
func (c Coordinator) CreateBlueprint(_ context.Context, req BlueprintCreateRequest) (*BlueprintCreateResult, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("blueprint name is required")
	}

	projectName := strings.TrimSpace(req.ProjectName)
	if projectName == "" {
		projectName = name
	}

	created := model.Blueprint{
		Name:        name,
		Version:     "v1",
		Description: strings.TrimSpace(req.Description),
		Runtime: model.BlueprintRuntime{
			ProjectName: projectName,
		},
		Services: []model.BlueprintService{},
	}
	if strings.TrimSpace(req.Middleware) != "" {
		service := model.BlueprintService{
			Name:       template.DefaultServiceName(req.Middleware, 1),
			Middleware: strings.TrimSpace(req.Middleware),
			Template:   strings.TrimSpace(req.Template),
			Values:     map[string]any{},
		}
		if strings.TrimSpace(req.Version) != "" {
			service.Values["version"] = strings.TrimSpace(req.Version)
		}
		created.Services = append(created.Services, service)
	}

	path := strings.TrimSpace(req.Path)
	if path == "" {
		root := strings.TrimSpace(req.Root)
		if root == "" {
			root = "."
		}
		path = filepath.Join(root, sanitizeBlueprintFileName(name)+".blueprint.yaml")
	}
	if err := c.blueprints.SaveBlueprint(path, created); err != nil {
		return nil, err
	}

	return &BlueprintCreateResult{
		Path:    filepath.Clean(path),
		Name:    created.Name,
		Message: "created blueprint " + filepath.Clean(path) + " for " + created.Name,
	}, nil
}

// DeleteBlueprint deletes one local blueprint file by path or name.
func (c Coordinator) DeleteBlueprint(ctx context.Context, ref string, root string) (*BlueprintDeleteResult, error) {
	resolved, err := c.ResolveBlueprint(ctx, ref, root)
	if err != nil {
		return nil, err
	}
	if err := c.blueprints.DeleteBlueprint(resolved.Path); err != nil {
		return nil, err
	}
	return &BlueprintDeleteResult{
		Path:    resolved.Path,
		Name:    resolved.Blueprint.Name,
		Message: "deleted blueprint " + resolved.Path + " for " + resolved.Blueprint.Name,
	}, nil
}

// UpdateBlueprint updates one local blueprint file by path or name.
func (c Coordinator) UpdateBlueprint(ctx context.Context, req BlueprintUpdateRequest) (*BlueprintUpdateResult, error) {
	resolved, err := c.ResolveBlueprint(ctx, req.Reference, req.Root)
	if err != nil {
		return nil, err
	}

	updated := resolved.Blueprint
	if name := strings.TrimSpace(req.Name); name != "" {
		updated.Name = name
	}
	if description := strings.TrimSpace(req.Description); description != "" {
		updated.Description = description
	}
	if projectName := strings.TrimSpace(req.ProjectName); projectName != "" {
		updated.Runtime.ProjectName = projectName
	}
	if err := applyBlueprintServiceUpdate(&updated, req); err != nil {
		return nil, err
	}

	if err := c.blueprints.UpdateBlueprint(resolved.Path, updated); err != nil {
		return nil, err
	}
	return &BlueprintUpdateResult{
		Path:    resolved.Path,
		Name:    updated.Name,
		Message: "updated blueprint " + resolved.Path + " for " + updated.Name,
	}, nil
}

// CopyBlueprint copies one existing blueprint into a new local blueprint file.
func (c Coordinator) CopyBlueprint(ctx context.Context, req BlueprintCopyRequest) (*BlueprintCopyResult, error) {
	resolved, err := c.ResolveBlueprint(ctx, req.Reference, req.Root)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("blueprint copy requires target name")
	}

	copied := resolved.Blueprint
	copied.Name = name
	if projectName := strings.TrimSpace(req.ProjectName); projectName != "" {
		copied.Runtime.ProjectName = projectName
	} else if copied.Runtime.ProjectName == "" {
		copied.Runtime.ProjectName = name
	}

	path := strings.TrimSpace(req.Path)
	if path == "" {
		root := strings.TrimSpace(req.Root)
		if root == "" {
			root = "."
		}
		path = filepath.Join(root, sanitizeBlueprintFileName(name)+".blueprint.yaml")
	}
	if err := c.blueprints.SaveBlueprint(path, copied); err != nil {
		return nil, err
	}
	return &BlueprintCopyResult{
		Path:    filepath.Clean(path),
		Name:    copied.Name,
		Message: "copied blueprint to " + filepath.Clean(path) + " for " + copied.Name,
	}, nil
}

func sanitizeBlueprintFileName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", "\\", "-")
	normalized = replacer.Replace(normalized)
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		return "zygarde"
	}
	return normalized
}

func applyBlueprintServiceUpdate(blueprint *model.Blueprint, req BlueprintUpdateRequest) error {
	if blueprint == nil {
		return fmt.Errorf("blueprint is required")
	}

	addServiceName := strings.TrimSpace(req.AddServiceName)
	removeServiceName := strings.TrimSpace(req.RemoveServiceName)
	targetServiceName := strings.TrimSpace(req.ServiceName)
	if addServiceName != "" && (removeServiceName != "" || targetServiceName != "") {
		return fmt.Errorf("blueprint update add-service cannot be combined with remove-service or service")
	}
	if removeServiceName != "" && targetServiceName != "" {
		return fmt.Errorf("blueprint update remove-service cannot be combined with service")
	}

	if addServiceName != "" {
		if req.Middleware == "" {
			return fmt.Errorf("blueprint update add-service requires middleware")
		}
		if findBlueprintService(*blueprint, addServiceName) >= 0 {
			return fmt.Errorf("blueprint service already exists: %s", addServiceName)
		}
		service := model.BlueprintService{
			Name:       addServiceName,
			Middleware: strings.TrimSpace(req.Middleware),
			Template:   strings.TrimSpace(req.Template),
			Values:     map[string]any{},
		}
		applyServiceValues(&service, req.SetValues)
		blueprint.Services = append(blueprint.Services, service)
		return nil
	}

	if removeServiceName != "" {
		index := findBlueprintService(*blueprint, removeServiceName)
		if index < 0 {
			return fmt.Errorf("blueprint service not found: %s", removeServiceName)
		}
		blueprint.Services = append(blueprint.Services[:index], blueprint.Services[index+1:]...)
		return nil
	}

	if targetServiceName == "" {
		return nil
	}
	index := findBlueprintService(*blueprint, targetServiceName)
	if index < 0 {
		return fmt.Errorf("blueprint service not found: %s", targetServiceName)
	}
	service := blueprint.Services[index]
	if middleware := strings.TrimSpace(req.Middleware); middleware != "" {
		service.Middleware = middleware
	}
	if templateName := strings.TrimSpace(req.Template); templateName != "" {
		service.Template = templateName
	}
	if service.Values == nil {
		service.Values = map[string]any{}
	}
	applyServiceValues(&service, req.SetValues)
	blueprint.Services[index] = service
	return nil
}

func findBlueprintService(blueprint model.Blueprint, name string) int {
	for index, service := range blueprint.Services {
		if service.Name == name {
			return index
		}
	}
	return -1
}

func applyServiceValues(service *model.BlueprintService, values map[string]string) {
	if service == nil || len(values) == 0 {
		return
	}
	if service.Values == nil {
		service.Values = map[string]any{}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		service.Values[key] = values[key]
	}
}
