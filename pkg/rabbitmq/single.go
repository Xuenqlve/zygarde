package rabbitmq

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
	middlewareName        = "rabbitmq"
	singleTemplate        = "single"
	defaultVersion        = "v4.2"
	defaultAMQPPort       = 5672
	defaultManagementPort = 15672
)

func init() {
	if err := Register(runtime.EnvironmentTypeCompose); err != nil {
		panic(err)
	}
}

// Register registers RabbitMQ specs into the provided registry.
func Register(envType runtime.EnvironmentType) error {
	if err := tpl.RegisterMiddleware(tpl.NewMiddlewareRuntimeKey(middlewareName, singleTemplate, envType), NewSingleSpec()); err != nil {
		return err
	}
	return tpl.RegisterMiddleware(tpl.NewMiddlewareRuntimeKey(middlewareName, clusterTemplate, envType), NewClusterSpec())
}

// NewSingleSpec returns the default RabbitMQ single-node middleware spec.
func NewSingleSpec() tpl.Middleware {
	return &singleSpec{}
}

type singleSpec struct {
	services []model.BlueprintService
}

func (*singleSpec) Middleware() string { return middlewareName }
func (*singleSpec) Template() string   { return singleTemplate }
func (*singleSpec) IsDefault() bool    { return true }

func (s *singleSpec) Normalize(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	name := input.Name
	if name == "" {
		name = tpl.DefaultServiceName(s.Middleware(), index)
	}

	values := tpl.MergeValues(s.DefaultValues(), input.Values)
	values[runtimecompose.ValueServiceName] = defaultStringValue(values[runtimecompose.ValueServiceName], name)
	values[runtimecompose.ValueContainerName] = defaultStringValue(values[runtimecompose.ValueContainerName], name)

	version := defaultStringValue(values[runtimecompose.ValueVersion], defaultVersion)
	if err := validateVersion(version); err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize rabbitmq single version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueDataDir] = defaultStringValue(values[runtimecompose.ValueDataDir], fmt.Sprintf("./data/%s", name))
	values[runtimecompose.ValueDefaultUser] = defaultStringValue(values[runtimecompose.ValueDefaultUser], "admin")
	values[runtimecompose.ValueDefaultPass] = defaultStringValue(values[runtimecompose.ValueDefaultPass], "admin123")
	values[runtimecompose.ValueErlangCookie] = defaultStringValue(values[runtimecompose.ValueErlangCookie], "rabbitmq-cookie")

	amqpPort, err := allocateOrReservePort(values[runtimecompose.ValueAMQPPort], hasValue(input.Values, runtimecompose.ValueAMQPPort), defaultAMQPPort, "rabbitmq single amqp_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueAMQPPort] = amqpPort

	managementPort, err := allocateOrReservePort(values[runtimecompose.ValueManagementPort], hasValue(input.Values, runtimecompose.ValueManagementPort), defaultManagementPort, "rabbitmq single management_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueManagementPort] = managementPort

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

		version := service.Values[runtimecompose.ValueVersion].(string)
		image := service.Values[runtimecompose.ValueImage].(string)
		containerName := service.Values[runtimecompose.ValueContainerName].(string)
		dataDir := service.Values[runtimecompose.ValueDataDir].(string)
		defaultUser := service.Values[runtimecompose.ValueDefaultUser].(string)
		defaultPass := service.Values[runtimecompose.ValueDefaultPass].(string)
		cookie := service.Values[runtimecompose.ValueErlangCookie].(string)
		amqpPort := service.Values[runtimecompose.ValueAMQPPort].(int)
		managementPort := service.Values[runtimecompose.ValueManagementPort].(int)
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
				Environment: map[string]string{
					"RABBITMQ_DEFAULT_USER":  defaultUser,
					"RABBITMQ_DEFAULT_PASS":  defaultPass,
					"RABBITMQ_ERLANG_COOKIE": cookie,
				},
				Ports: []runtime.PortBinding{
					{HostPort: amqpPort, ContainerPort: 5672, Protocol: "tcp"},
					{HostPort: managementPort, ContainerPort: 15672, Protocol: "tcp"},
				},
				Volumes: []runtime.VolumeMount{
					{Source: dataDir, Target: "/var/lib/rabbitmq"},
				},
				HealthCheck: &runtime.HealthCheck{
					Test:        []string{"CMD-SHELL", "rabbitmq-diagnostics -q ping"},
					Interval:    "5s",
					Timeout:     "5s",
					Retries:     40,
					StartPeriod: "20s",
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "rabbitmq-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"%s_VERSION=%s\n%s_AMQP_PORT=%d\n%s_MANAGEMENT_PORT=%d\n%s_DEFAULT_USER=%s\n%s_DEFAULT_PASS=%s\n%s_ERLANG_COOKIE=%s\n",
						envKeyPrefix, version,
						envKeyPrefix, amqpPort,
						envKeyPrefix, managementPort,
						envKeyPrefix, defaultUser,
						envKeyPrefix, defaultPass,
						envKeyPrefix, cookie,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "rabbitmq-build",
					PathKey:   "build_script",
					Content:   fmt.Sprintf("echo \"RabbitMQ %s (%s) compose stack started\"\n", service.Name, version),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:    "rabbitmq-check",
					PathKey: "check_script",
					Content: fmt.Sprintf(
						"\"$CONTAINER_ENGINE\" exec %s rabbitmq-diagnostics -q ping\n\"$CONTAINER_ENGINE\" exec %s rabbitmqctl status | grep -E \"RabbitMQ version|Cluster name|Uptime\" || true\n",
						containerName, containerName,
					),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "rabbitmq-readme",
					PathKey:   "readme_file",
					Content:   fmt.Sprintf("# RabbitMQ %s\n\n- version: %s\n- image: %s\n- amqp port: %d\n- management port: %d\n", service.Name, version, image, amqpPort, managementPort),
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
		return fmt.Errorf("rabbitmq single validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != singleTemplate {
		return fmt.Errorf("rabbitmq single validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("rabbitmq single validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("rabbitmq single validate version: %w", err)
	}

	amqpPort, err := normalizePort(service.Values[runtimecompose.ValueAMQPPort])
	if err != nil {
		return fmt.Errorf("rabbitmq single validate amqp_port: %w", err)
	}
	managementPort, err := normalizePort(service.Values[runtimecompose.ValueManagementPort])
	if err != nil {
		return fmt.Errorf("rabbitmq single validate management_port: %w", err)
	}
	if amqpPort <= 0 || managementPort <= 0 {
		return fmt.Errorf("rabbitmq single validate ports: must be greater than 0")
	}
	if amqpPort == managementPort {
		return fmt.Errorf("rabbitmq single validate ports: amqp_port and management_port must be different")
	}

	fields := []struct{ key, name string }{
		{runtimecompose.ValueServiceName, "service_name"},
		{runtimecompose.ValueContainerName, "container_name"},
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueDataDir, "data_dir"},
		{runtimecompose.ValueDefaultUser, "default_user"},
		{runtimecompose.ValueDefaultPass, "default_pass"},
		{runtimecompose.ValueErlangCookie, "erlang_cookie"},
	}
	for _, field := range fields {
		value, ok := service.Values[field.key].(string)
		if !ok || value == "" {
			return fmt.Errorf("rabbitmq single validate %s: must be a non-empty string", field.name)
		}
	}
	return nil
}

func (*singleSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:        defaultVersion,
		runtimecompose.ValueImage:          "",
		runtimecompose.ValueDataDir:        "",
		runtimecompose.ValueAMQPPort:       defaultAMQPPort,
		runtimecompose.ValueManagementPort: defaultManagementPort,
		runtimecompose.ValueDefaultUser:    "admin",
		runtimecompose.ValueDefaultPass:    "admin123",
		runtimecompose.ValueErlangCookie:   "rabbitmq-cookie",
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

func allocateOrReservePort(value any, userSpecified bool, fallback int, fieldName string) (int, error) {
	if userSpecified {
		port, err := normalizePort(value)
		if err != nil {
			return 0, fmt.Errorf("normalize %s: %w", fieldName, err)
		}
		if err := tool.ReservePort(port); err != nil {
			return 0, fmt.Errorf("normalize %s: %w", fieldName, err)
		}
		return port, nil
	}

	port, err := tool.AllocatePort(fallback)
	if err != nil {
		return 0, fmt.Errorf("normalize %s: %w", fieldName, err)
	}
	return port, nil
}

func imageForVersion(version string) string {
	if version == "v4.2" {
		return "rabbitmq:4.2-management"
	}
	return "rabbitmq:4.2-management"
}

func validateVersion(version string) error {
	if version != "v4.2" {
		return fmt.Errorf("unsupported version %q", version)
	}
	return nil
}

func serviceEnvKeyPrefix(name string) string {
	normalized := strings.ToUpper(name)
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return "RABBITMQ_" + normalized
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
