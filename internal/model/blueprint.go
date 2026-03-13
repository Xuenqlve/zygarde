package model

// Blueprint describes one user-facing environment definition file.
type Blueprint struct {
	Name        string
	Version     string
	Description string
	Runtime     BlueprintRuntime
	Services    []BlueprintService
}

// BlueprintRuntime contains optional runtime overrides for one environment.
type BlueprintRuntime struct {
	ProjectName string
	AutoRemove  bool
}

// BlueprintService describes one normalized middleware instance in a blueprint.
type BlueprintService struct {
	Name       string
	Middleware string
	Template   string
	Values     map[string]any
}
