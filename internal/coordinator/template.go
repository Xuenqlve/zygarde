package coordinator

import (
	"context"
	"fmt"
	"strings"

	"github.com/xuenqlve/zygarde/pkg/catalog"
)

// TemplateListItem captures one built-in template capability.
type TemplateListItem struct {
	Middleware  string
	Template    string
	RuntimeType string
	Description string
	Default     bool
	Versions    []string
	DocPath     string
}

// TemplateListResult contains all discovered built-in template capabilities.
type TemplateListResult struct {
	Items []TemplateListItem
}

// TemplateShowResult contains one built-in template capability detail.
type TemplateShowResult struct {
	Middleware  string
	Template    string
	RuntimeType string
	Description string
	Default     bool
	Versions    []string
	DocPath     string
}

// ListTemplates lists all built-in template capabilities.
func (c Coordinator) ListTemplates(_ context.Context, runtimeType string) (*TemplateListResult, error) {
	items := catalog.ListTemplates()
	result := &TemplateListResult{Items: make([]TemplateListItem, 0, len(items))}
	for _, item := range items {
		if runtimeType != "" && item.RuntimeType != runtimeType {
			continue
		}
		result.Items = append(result.Items, TemplateListItem{
			Middleware:  item.Middleware,
			Template:    item.Template,
			RuntimeType: item.RuntimeType,
			Description: item.Description,
			Default:     item.Default,
			Versions:    append([]string(nil), item.Versions...),
			DocPath:     item.DocPath,
		})
	}
	return result, nil
}

// ShowTemplate returns one built-in template capability detail.
func (c Coordinator) ShowTemplate(_ context.Context, middleware, templateName, runtimeType string) (*TemplateShowResult, error) {
	info, ok := catalog.GetTemplate(middleware, templateName, runtimeType)
	if !ok {
		return nil, fmt.Errorf("template not found: %s/%s/%s", middleware, templateName, runtimeType)
	}
	return &TemplateShowResult{
		Middleware:  info.Middleware,
		Template:    info.Template,
		RuntimeType: info.RuntimeType,
		Description: info.Description,
		Default:     info.Default,
		Versions:    append([]string(nil), info.Versions...),
		DocPath:     info.DocPath,
	}, nil
}

// SplitTemplateReference parses a template reference like mysql/single.
func SplitTemplateReference(value string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("template reference must be <middleware>/<template>")
	}
	return parts[0], parts[1], nil
}
