package mysql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

const (
	middlewareName = "mysql"
	singleTemplate = "single"
	defaultPort    = 3306
	defaultVersion = "v8.0"
)

func init() {
	if err := Register(runtime.EnvironmentTypeCompose); err != nil {
		panic(err)
	}
}

// Register registers MySQL specs into the provided registry.
func Register(envType runtime.EnvironmentType) error {
	return tpl.RegisterMiddleware(tpl.NewMiddlewareRuntimeKey(middlewareName, singleTemplate, envType), NewSingleSpec())
}

// NewSingleSpec returns the default MySQL single-node middleware spec.
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

	values := tpl.MergeValues(s.DefaultValues(), input.Values)
	values[runtimecompose.ValueServiceName] = defaultStringValue(values[runtimecompose.ValueServiceName], name)
	values[runtimecompose.ValueContainerName] = defaultStringValue(values[runtimecompose.ValueContainerName], name)
	version := defaultStringValue(values[runtimecompose.ValueVersion], defaultVersion)
	if err := validateVersion(version); err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize mysql single version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueDataDir] = defaultStringValue(values[runtimecompose.ValueDataDir], fmt.Sprintf("./data/%s", name))

	port, err := normalizePort(values[runtimecompose.ValuePort])
	if err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize mysql single port: %w", err)
	}
	values[runtimecompose.ValuePort] = port

	rootPassword, ok := values[runtimecompose.ValueRootPassword].(string)
	if !ok || rootPassword == "" {
		return model.BlueprintService{}, fmt.Errorf("normalize mysql single root_password: must be a non-empty string")
	}

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
	contexts := make([]runtime.EnvironmentContext, 0, len(s.services))
	for _, service := range s.services {
		if err := s.Validate(service); err != nil {
			return nil, err
		}

		port, err := normalizePort(service.Values[runtimecompose.ValuePort])
		if err != nil {
			return nil, fmt.Errorf("mysql single build runtime context port: %w", err)
		}

		rootPassword := service.Values[runtimecompose.ValueRootPassword].(string)
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
				Platform:      platformForVersion(version),
				ContainerName: containerName,
				Restart:       "unless-stopped",
				Environment: map[string]string{
					"MYSQL_ROOT_PASSWORD": rootPassword,
					"MYSQL_ROOT_HOST":     "%",
				},
				Ports: []runtime.PortBinding{
					{
						HostPort:      port,
						ContainerPort: 3306,
						Protocol:      "tcp",
					},
				},
				Volumes: []runtime.VolumeMount{
					{
						Source: dataDir,
						Target: "/var/lib/mysql",
					},
				},
				Command: commandForVersion(version),
				HealthCheck: &runtime.HealthCheck{
					Test: []string{
						"CMD",
						"mysqladmin",
						"ping",
						"-h",
						"127.0.0.1",
						"-uroot",
						"-p" + rootPassword,
					},
					Interval:    "5s",
					Timeout:     "5s",
					Retries:     30,
					StartPeriod: "20s",
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "mysql-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"%s_VERSION=%s\n%s_IMAGE=%s\n%s_PORT=%d\n%s_ROOT_PASSWORD=%s\n",
						envKeyPrefix,
						version,
						envKeyPrefix,
						image,
						envKeyPrefix,
						port,
						envKeyPrefix,
						rootPassword,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "mysql-build",
					PathKey:   "build_script",
					Content:   fmt.Sprintf("echo \"MySQL %s (%s) compose stack started\"\n", service.Name, version),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:    "mysql-check",
					PathKey: "check_script",
					Content: fmt.Sprintf(
						"docker exec %s mysql -uroot \"-p${%s_ROOT_PASSWORD}\" -e \"SELECT 1;\"\n",
						containerName,
						envKeyPrefix,
					),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "mysql-readme",
					PathKey:   "readme_file",
					Content:   fmt.Sprintf("# MySQL %s\n\n- version: %s\n- image: %s\n- port: %d\n", service.Name, version, image, port),
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
		return fmt.Errorf("mysql single validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != singleTemplate {
		return fmt.Errorf("mysql single validate: unexpected template %q", service.Template)
	}

	port, err := normalizePort(service.Values[runtimecompose.ValuePort])
	if err != nil {
		return fmt.Errorf("mysql single validate port: %w", err)
	}
	if port <= 0 {
		return fmt.Errorf("mysql single validate port: must be greater than 0")
	}

	rootPassword, ok := service.Values[runtimecompose.ValueRootPassword].(string)
	if !ok || rootPassword == "" {
		return fmt.Errorf("mysql single validate root_password: must be a non-empty string")
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("mysql single validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("mysql single validate version: %w", err)
	}

	serviceName, ok := service.Values[runtimecompose.ValueServiceName].(string)
	if !ok || serviceName == "" {
		return fmt.Errorf("mysql single validate service_name: must be a non-empty string")
	}

	containerName, ok := service.Values[runtimecompose.ValueContainerName].(string)
	if !ok || containerName == "" {
		return fmt.Errorf("mysql single validate container_name: must be a non-empty string")
	}

	image, ok := service.Values[runtimecompose.ValueImage].(string)
	if !ok || image == "" {
		return fmt.Errorf("mysql single validate image: must be a non-empty string")
	}

	dataDir, ok := service.Values[runtimecompose.ValueDataDir].(string)
	if !ok || dataDir == "" {
		return fmt.Errorf("mysql single validate data_dir: must be a non-empty string")
	}

	return nil
}

func defaultStringValue(value any, fallback string) string {
	current, ok := value.(string)
	if !ok || current == "" {
		return fallback
	}
	return current
}

func (*singleSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValuePort:         defaultPort,
		runtimecompose.ValueRootPassword: "root",
		runtimecompose.ValueVersion:      defaultVersion,
		runtimecompose.ValueImage:        "",
		runtimecompose.ValueDataDir:      "",
	}
}

func imageForVersion(version string) string {
	switch version {
	case "v5.7":
		return "mysql:5.7"
	case "v8.0":
		return "mysql:8.0"
	default:
		return "mysql:8.0"
	}
}

func commandForVersion(version string) []string {
	command := []string{
		"--server-id=1",
		"--log-bin=mysql-bin",
		"--binlog-format=ROW",
		"--gtid-mode=ON",
		"--enforce-gtid-consistency=ON",
		"--skip-name-resolve=1",
	}
	if version == "v5.7" {
		command = append(command, "--default-authentication-plugin=mysql_native_password")
	}
	return command
}

func validateVersion(version string) error {
	switch version {
	case "v5.7", "v8.0":
		return nil
	default:
		return fmt.Errorf("unsupported version %q", version)
	}
}

func platformForVersion(version string) string {
	if version == "v5.7" {
		return "linux/amd64"
	}
	return ""
}

func serviceEnvKeyPrefix(name string) string {
	normalized := strings.ToUpper(name)
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return "MYSQL_" + normalized
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
		parsed, err := strconv.Atoi(port)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported port type %T", value)
	}
}
