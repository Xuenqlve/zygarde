package model

// Blueprint describes one user-facing environment definition file.
type Blueprint struct {
	Name        string             `yaml:"name"`
	Version     string             `yaml:"version"`
	Description string             `yaml:"description"`
	Runtime     BlueprintRuntime   `yaml:"runtime"`
	Services    []BlueprintService `yaml:"services"`
}

// BlueprintRuntime contains optional runtime overrides for one environment.
type BlueprintRuntime struct {
	ProjectName string `yaml:"project-name"`
	AutoRemove  bool   `yaml:"auto-remove"`
}

// BlueprintService describes one normalized middleware instance in a blueprint.
type BlueprintService struct {
	Name       string         `yaml:"name"`
	Middleware string         `yaml:"middleware"`
	Template   string         `yaml:"template"`
	Values     map[string]any `yaml:"values"`
}
