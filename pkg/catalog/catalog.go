package catalog

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// TemplateInfo describes one built-in middleware template capability exposed from pkg.
type TemplateInfo struct {
	Middleware  string
	Template    string
	RuntimeType string
	Description string
	Default     bool
	Versions    []string
	DocPath     string
}

var (
	templateRegistry      map[string]TemplateInfo
	templateRegistryMutex sync.Mutex
)

// RegisterTemplate registers one built-in template capability.
func RegisterTemplate(info TemplateInfo) error {
	templateRegistryMutex.Lock()
	defer templateRegistryMutex.Unlock()

	if strings.TrimSpace(info.Middleware) == "" {
		return fmt.Errorf("template metadata middleware is required")
	}
	if strings.TrimSpace(info.Template) == "" {
		return fmt.Errorf("template metadata template is required")
	}
	if strings.TrimSpace(info.RuntimeType) == "" {
		return fmt.Errorf("template metadata runtime type is required")
	}

	if templateRegistry == nil {
		templateRegistry = make(map[string]TemplateInfo)
	}
	key := registryKey(info.Middleware, info.Template, info.RuntimeType)
	if _, exists := templateRegistry[key]; exists {
		return fmt.Errorf("template metadata already registered: %s", key)
	}

	copied := info
	copied.Versions = append([]string(nil), info.Versions...)
	templateRegistry[key] = copied
	return nil
}

// ListTemplates returns all registered template capabilities.
func ListTemplates() []TemplateInfo {
	templateRegistryMutex.Lock()
	defer templateRegistryMutex.Unlock()

	items := make([]TemplateInfo, 0, len(templateRegistry))
	for _, info := range templateRegistry {
		copied := info
		copied.Versions = append([]string(nil), info.Versions...)
		items = append(items, copied)
	}
	sort.Slice(items, func(i, j int) bool {
		left := items[i].Middleware + "/" + items[i].Template + "/" + items[i].RuntimeType
		right := items[j].Middleware + "/" + items[j].Template + "/" + items[j].RuntimeType
		return left < right
	})
	return items
}

// GetTemplate returns one registered template capability.
func GetTemplate(middleware, template, runtimeType string) (TemplateInfo, bool) {
	templateRegistryMutex.Lock()
	defer templateRegistryMutex.Unlock()

	info, ok := templateRegistry[registryKey(middleware, template, runtimeType)]
	if !ok {
		return TemplateInfo{}, false
	}
	info.Versions = append([]string(nil), info.Versions...)
	return info, true
}

func registryKey(middleware, template, runtimeType string) string {
	return middleware + "/" + template + "/" + runtimeType
}
