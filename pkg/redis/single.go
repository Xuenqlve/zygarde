package redis

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
	"github.com/xuenqlve/zygarde/internal/tool"
)

const (
	middlewareName = "redis"
	singleTemplate = "single"
	defaultPort    = 6379
	defaultVersion = "v7.4"
)

func init() {
	if err := Register(runtime.EnvironmentTypeCompose); err != nil {
		panic(err)
	}
}

// Register registers Redis specs into the provided registry.
func Register(envType runtime.EnvironmentType) error {
	if err := tpl.RegisterMiddleware(tpl.NewMiddlewareRuntimeKey(middlewareName, singleTemplate, envType), NewSingleSpec()); err != nil {
		return err
	}
	if err := tpl.RegisterMiddleware(tpl.NewMiddlewareRuntimeKey(middlewareName, masterSlaveTemplate, envType), NewMasterSlaveSpec()); err != nil {
		return err
	}
	return tpl.RegisterMiddleware(tpl.NewMiddlewareRuntimeKey(middlewareName, clusterTemplate, envType), NewClusterSpec())
}

// NewSingleSpec returns the default Redis single-node middleware spec.
func NewSingleSpec() tpl.Middleware {
	return &singleSpec{}
}

type singleSpec struct {
	services []model.BlueprintService
}

func (*singleSpec) Middleware() string {
	return middlewareName
}

func (*singleSpec) Template() string {
	return singleTemplate
}

func (*singleSpec) IsDefault() bool {
	return true
}

func (s *singleSpec) Normalize(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	name := input.Name
	if name == "" {
		name = tpl.DefaultServiceName(s.Middleware(), index)
	}

	userSpecifiedPort := hasValue(input.Values, runtimecompose.ValuePort)
	values := tpl.MergeValues(s.DefaultValues(), input.Values)
	values[runtimecompose.ValueServiceName] = defaultStringValue(values[runtimecompose.ValueServiceName], name)
	values[runtimecompose.ValueContainerName] = defaultStringValue(values[runtimecompose.ValueContainerName], name)

	version := defaultStringValue(values[runtimecompose.ValueVersion], defaultVersion)
	if err := validateVersion(version); err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize redis single version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueDataDir] = defaultStringValue(values[runtimecompose.ValueDataDir], fmt.Sprintf("./data/%s", name))

	var (
		port int
		err  error
	)
	if userSpecifiedPort {
		port, err = normalizePort(values[runtimecompose.ValuePort])
		if err != nil {
			return model.BlueprintService{}, fmt.Errorf("normalize redis single port: %w", err)
		}
		if err := tool.ReservePort(port); err != nil {
			return model.BlueprintService{}, fmt.Errorf("normalize redis single port: %w", err)
		}
	} else {
		port, err = tool.AllocatePort(defaultPort)
		if err != nil {
			return model.BlueprintService{}, fmt.Errorf("normalize redis single port: %w", err)
		}
	}
	values[runtimecompose.ValuePort] = port

	return model.BlueprintService{
		Name:       name,
		Middleware: s.Middleware(),
		Template:   s.Template(),
		Values:     values,
	}, nil
}

func (s *singleSpec) Configure(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	service, err := s.Normalize(input, index)
	if err != nil {
		return model.BlueprintService{}, err
	}
	if err := s.Validate(service); err != nil {
		return model.BlueprintService{}, err
	}
	s.services = append(s.services, service)
	return service, nil
}

func (s *singleSpec) BuildRuntimeContexts(runtimeType runtime.EnvironmentType) ([]runtime.EnvironmentContext, error) {
	services := append([]model.BlueprintService(nil), s.services...)
	s.services = nil

	contexts := make([]runtime.EnvironmentContext, 0, len(services))
	for _, service := range services {
		if err := s.Validate(service); err != nil {
			return nil, err
		}

		port, err := normalizePort(service.Values[runtimecompose.ValuePort])
		if err != nil {
			return nil, fmt.Errorf("redis single build runtime context port: %w", err)
		}

		version := service.Values[runtimecompose.ValueVersion].(string)
		containerName := service.Values[runtimecompose.ValueContainerName].(string)
		image := service.Values[runtimecompose.ValueImage].(string)
		dataDir := service.Values[runtimecompose.ValueDataDir].(string)
		envKeyPrefix := serviceEnvKeyPrefix(service.Name)

		contexts = append(contexts, runtime.ComposeContext{
			EnvType:     runtimeType,
			ServiceName: service.Name,
			Middleware:  service.Middleware,
			Template:    service.Template,
			Service: runtime.ServiceSpec{
				Image:         image,
				ContainerName: containerName,
				Restart:       "unless-stopped",
				Ports: []runtime.PortBinding{
					{
						HostPort:      port,
						ContainerPort: 6379,
						Protocol:      "tcp",
					},
				},
				Volumes: []runtime.VolumeMount{
					{
						Source: dataDir,
						Target: "/data",
					},
				},
				Command: []string{
					"redis-server",
					"--appendonly",
					"yes",
					"--save",
					"60 1000",
				},
				HealthCheck: &runtime.HealthCheck{
					Test:        []string{"CMD", "redis-cli", "ping"},
					Interval:    "5s",
					Timeout:     "5s",
					Retries:     30,
					StartPeriod: "10s",
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "redis-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"%s_VERSION=%s\n%s_IMAGE=%s\n%s_PORT=%d\n",
						envKeyPrefix,
						version,
						envKeyPrefix,
						image,
						envKeyPrefix,
						port,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "redis-build",
					PathKey:   "build_script",
					Content:   fmt.Sprintf("echo \"Redis %s (%s) compose stack started\"\n", service.Name, version),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:    "redis-check",
					PathKey: "check_script",
					Content: fmt.Sprintf(
						"\"$CONTAINER_ENGINE\" exec %s redis-cli ping\n\"$CONTAINER_ENGINE\" exec %s redis-cli info replication | grep '^role:'\n",
						containerName,
						containerName,
					),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "redis-readme",
					PathKey:   "readme_file",
					Content:   fmt.Sprintf("# Redis %s\n\n- version: %s\n- image: %s\n- port: %d\n", service.Name, version, image, port),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeReadme,
				},
			},
			Metadata: tpl.MergeValues(nil, service.Values),
		})
	}

	return contexts, nil
}

func (*singleSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template == "" {
		return tpl.ErrServiceTemplateRequired
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("redis single validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != singleTemplate {
		return fmt.Errorf("redis single validate: unexpected template %q", service.Template)
	}

	port, err := normalizePort(service.Values[runtimecompose.ValuePort])
	if err != nil {
		return fmt.Errorf("redis single validate port: %w", err)
	}
	if port <= 0 {
		return fmt.Errorf("redis single validate port: must be greater than 0")
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("redis single validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("redis single validate version: %w", err)
	}

	stringFields := []struct {
		key  string
		name string
	}{
		{runtimecompose.ValueServiceName, "service_name"},
		{runtimecompose.ValueContainerName, "container_name"},
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueDataDir, "data_dir"},
	}
	for _, field := range stringFields {
		value, ok := service.Values[field.key].(string)
		if !ok || value == "" {
			return fmt.Errorf("redis single validate %s: must be a non-empty string", field.name)
		}
	}

	return nil
}

func (*singleSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValuePort:    defaultPort,
		runtimecompose.ValueVersion: defaultVersion,
		runtimecompose.ValueImage:   "",
		runtimecompose.ValueDataDir: "",
	}
}

func defaultStringValue(value any, fallback string) string {
	current, ok := value.(string)
	if !ok || current == "" {
		return fallback
	}
	return current
}

func hasValue(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	_, ok := values[key]
	return ok
}

func imageForVersion(version string) string {
	switch version {
	case "v6.2":
		return "redis:6.2"
	case "v7.4":
		return "redis:7.4"
	default:
		return "redis:7.4"
	}
}

func validateVersion(version string) error {
	switch version {
	case "v6.2", "v7.4":
		return nil
	default:
		return fmt.Errorf("unsupported version %q", version)
	}
}

func serviceEnvKeyPrefix(name string) string {
	normalized := strings.ToUpper(name)
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return "REDIS_" + normalized
}

func normalizePort(value any) (int, error) {
	switch port := value.(type) {
	case int:
		return port, nil
	case int8:
		return int(port), nil
	case int16:
		return int(port), nil
	case int32:
		return int(port), nil
	case int64:
		return int(port), nil
	case uint:
		return int(port), nil
	case uint8:
		return int(port), nil
	case uint16:
		return int(port), nil
	case uint32:
		return int(port), nil
	case uint64:
		return int(port), nil
	case float32:
		return int(port), nil
	case float64:
		return int(port), nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(port))
		if err != nil {
			return 0, fmt.Errorf("parse port %q: %w", port, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported port type %T", value)
	}
}
