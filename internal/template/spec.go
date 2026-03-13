package template

import "github.com/xuenqlve/zygarde/internal/model"

// ServiceInput describes the user-facing service block from blueprint.yaml.
type ServiceInput struct {
	Name       string
	Middleware string
	Template   string
	Values     map[string]any
}

// ServiceSpec defines one unique middleware and template implementation.
type ServiceSpec interface {
	Middleware() string
	Template() string
	IsDefault() bool
	DefaultValues() map[string]any
	Normalize(input ServiceInput, index int) (model.BlueprintService, error)
	Validate(service model.BlueprintService) error
}

// Registry manages service specs registered by middleware packages.
type Registry interface {
	Register(spec ServiceSpec) error
	Get(middleware, template string) (ServiceSpec, bool)
	Default(middleware string) (ServiceSpec, bool)
	List() []ServiceSpec
}
