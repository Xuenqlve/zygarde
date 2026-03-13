package runtime

// EnvironmentType identifies one runtime backend.
type EnvironmentType string

const (
	EnvironmentTypeCompose EnvironmentType = "compose"
	EnvironmentTypeK8s     EnvironmentType = "k8s"
)

// EnvironmentContext is the normalized runtime input shared with deployment layers.
type EnvironmentContext struct {
	RuntimeType EnvironmentType
	ServiceName string
	Middleware  string
	Template    string
	Values      map[string]any
}
