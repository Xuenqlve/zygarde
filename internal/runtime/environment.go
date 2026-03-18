package runtime

// EnvironmentType identifies one runtime backend.
type EnvironmentType string

const (
	EnvironmentTypeCompose EnvironmentType = "compose"
	EnvironmentTypeK8s     EnvironmentType = "k8s"
)

// RenderInput is the incremental input contract for the render stage.
type RenderInput struct {
	ServiceName string
	Middleware  string
	Template    string
	Service     ServiceSpec
	Assets      []AssetSpec
}

// ApplyInput is the incremental input contract for the apply stage.
type ApplyInput struct {
	ServiceName string
	Middleware  string
	Template    string
}

// EnvironmentContext is the shared runtime context contract used to wire the create flow.
type EnvironmentContext interface {
	RuntimeType() EnvironmentType
	PrepareInput() PrepareInput
	RenderInput() RenderInput
	ApplyInput() ApplyInput
}

// ComposeContext is the current compose runtime context implementation.
type ComposeContext struct {
	EnvType     EnvironmentType
	ServiceName string
	Middleware  string
	Template    string
	Service     ServiceSpec
	Assets      []AssetSpec
	Metadata    map[string]any
}

// RuntimeType returns the runtime backend associated with the context.
func (c ComposeContext) RuntimeType() EnvironmentType {
	return c.EnvType
}

// PrepareInput returns the current prepare-stage input placeholder.
func (ComposeContext) PrepareInput() PrepareInput {
	return PrepareInput{}
}

// RenderInput returns the current render-stage input placeholder.
func (c ComposeContext) RenderInput() RenderInput {
	return RenderInput{
		ServiceName: c.ServiceName,
		Middleware:  c.Middleware,
		Template:    c.Template,
		Service:     c.Service,
		Assets:      append([]AssetSpec(nil), c.Assets...),
	}
}

// ApplyInput returns the current apply-stage input placeholder.
func (c ComposeContext) ApplyInput() ApplyInput {
	return ApplyInput{
		ServiceName: c.ServiceName,
		Middleware:  c.Middleware,
		Template:    c.Template,
	}
}

// ServiceSpec describes one runtime-ready service definition.
type ServiceSpec struct {
	Image         string
	Platform      string
	Hostname      string
	ContainerName string
	Restart       string
	Environment   map[string]string
	Ports         []PortBinding
	Volumes       []VolumeMount
	Command       []string
	HealthCheck   *HealthCheck
}

// PortBinding maps one host port to one container port.
type PortBinding struct {
	HostPort      int
	ContainerPort int
	Protocol      string
}

// VolumeMount maps one source path into the container filesystem.
type VolumeMount struct {
	Source   string
	Target   string
	ReadOnly bool
}

// HealthCheck describes one runtime health check.
type HealthCheck struct {
	Test        []string
	Interval    string
	Timeout     string
	Retries     int
	StartPeriod string
}

// AssetMergeMode defines how render should merge same-target assets.
type AssetMergeMode string

const (
	AssetMergeEnv    AssetMergeMode = "env"
	AssetMergeScript AssetMergeMode = "script"
	AssetMergeReadme AssetMergeMode = "readme"
	AssetMergeUnique AssetMergeMode = "unique"
)

// AssetSpec describes one runtime asset contribution from a context.
type AssetSpec struct {
	Name      string
	PathKey   string
	FileName  string
	Content   string
	Mode      int
	MergeMode AssetMergeMode
}
