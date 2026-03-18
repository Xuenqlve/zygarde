package postgresql

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
	middlewareName = "postgresql"
	singleTemplate = "single"
	defaultPort    = 5432
	defaultVersion = "v17"
)

func init() {
	if err := Register(runtime.EnvironmentTypeCompose); err != nil {
		panic(err)
	}
}

// Register registers PostgreSQL specs into the provided registry.
func Register(envType runtime.EnvironmentType) error {
	if err := tpl.RegisterMiddleware(tpl.NewMiddlewareRuntimeKey(middlewareName, singleTemplate, envType), NewSingleSpec()); err != nil {
		return err
	}
	return tpl.RegisterMiddleware(tpl.NewMiddlewareRuntimeKey(middlewareName, masterSlaveTemplate, envType), NewMasterSlaveSpec())
}

// NewSingleSpec returns the default PostgreSQL single-node middleware spec.
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

	userSpecifiedPort := hasValue(input.Values, runtimecompose.ValuePort)
	values := tpl.MergeValues(s.DefaultValues(), input.Values)
	values[runtimecompose.ValueServiceName] = defaultStringValue(values[runtimecompose.ValueServiceName], name)
	values[runtimecompose.ValueContainerName] = defaultStringValue(values[runtimecompose.ValueContainerName], name)

	version := defaultStringValue(values[runtimecompose.ValueVersion], defaultVersion)
	if err := validateVersion(version); err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize postgresql single version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueDataDir] = defaultStringValue(values[runtimecompose.ValueDataDir], fmt.Sprintf("./data/%s", name))
	values[runtimecompose.ValueUser] = defaultStringValue(values[runtimecompose.ValueUser], "postgres")
	values[runtimecompose.ValuePassword] = defaultStringValue(values[runtimecompose.ValuePassword], "postgres123")
	values[runtimecompose.ValueDatabase] = defaultStringValue(values[runtimecompose.ValueDatabase], "app")

	var (
		port int
		err  error
	)
	if userSpecifiedPort {
		port, err = normalizePort(values[runtimecompose.ValuePort])
		if err != nil {
			return model.BlueprintService{}, fmt.Errorf("normalize postgresql single port: %w", err)
		}
		if err := tool.ReservePort(port); err != nil {
			return model.BlueprintService{}, fmt.Errorf("normalize postgresql single port: %w", err)
		}
	} else {
		port, err = tool.AllocatePort(defaultPort)
		if err != nil {
			return model.BlueprintService{}, fmt.Errorf("normalize postgresql single port: %w", err)
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
			return nil, fmt.Errorf("postgresql single build runtime context port: %w", err)
		}

		version := service.Values[runtimecompose.ValueVersion].(string)
		containerName := service.Values[runtimecompose.ValueContainerName].(string)
		image := service.Values[runtimecompose.ValueImage].(string)
		dataDir := service.Values[runtimecompose.ValueDataDir].(string)
		user := service.Values[runtimecompose.ValueUser].(string)
		password := service.Values[runtimecompose.ValuePassword].(string)
		database := service.Values[runtimecompose.ValueDatabase].(string)
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
					"POSTGRES_USER":     user,
					"POSTGRES_PASSWORD": password,
					"POSTGRES_DB":       database,
				},
				Ports: []runtime.PortBinding{
					{HostPort: port, ContainerPort: 5432, Protocol: "tcp"},
				},
				Volumes: []runtime.VolumeMount{
					{Source: dataDir, Target: "/var/lib/postgresql/data"},
				},
				HealthCheck: &runtime.HealthCheck{
					Test:        []string{"CMD-SHELL", fmt.Sprintf("pg_isready -U %s -d %s", user, database)},
					Interval:    "5s",
					Timeout:     "5s",
					Retries:     30,
					StartPeriod: "20s",
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "postgres-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"%s_VERSION=%s\n%s_IMAGE=%s\n%s_PORT=%d\n%s_USER=%s\n%s_PASSWORD=%s\n%s_DATABASE=%s\n",
						envKeyPrefix, version,
						envKeyPrefix, image,
						envKeyPrefix, port,
						envKeyPrefix, user,
						envKeyPrefix, password,
						envKeyPrefix, database,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "postgres-build",
					PathKey:   "build_script",
					Content:   fmt.Sprintf("echo \"PostgreSQL %s (%s) compose stack started\"\n", service.Name, version),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:    "postgres-check",
					PathKey: "check_script",
					Content: fmt.Sprintf(
						"\"$CONTAINER_ENGINE\" exec %s psql -U \"${%s_USER}\" -d \"${%s_DATABASE}\" -c 'select 1;'\n\"$CONTAINER_ENGINE\" exec %s psql -U \"${%s_USER}\" -d \"${%s_DATABASE}\" -tAc 'select version();'\n",
						containerName, envKeyPrefix, envKeyPrefix,
						containerName, envKeyPrefix, envKeyPrefix,
					),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "postgres-readme",
					PathKey:   "readme_file",
					Content:   fmt.Sprintf("# PostgreSQL %s\n\n- version: %s\n- image: %s\n- port: %d\n- user: %s\n- database: %s\n", service.Name, version, image, port, user, database),
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
		return fmt.Errorf("postgresql single validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != singleTemplate {
		return fmt.Errorf("postgresql single validate: unexpected template %q", service.Template)
	}

	port, err := normalizePort(service.Values[runtimecompose.ValuePort])
	if err != nil {
		return fmt.Errorf("postgresql single validate port: %w", err)
	}
	if port <= 0 {
		return fmt.Errorf("postgresql single validate port: must be greater than 0")
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("postgresql single validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("postgresql single validate version: %w", err)
	}

	fields := []struct{ key, name string }{
		{runtimecompose.ValueServiceName, "service_name"},
		{runtimecompose.ValueContainerName, "container_name"},
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueDataDir, "data_dir"},
		{runtimecompose.ValueUser, "user"},
		{runtimecompose.ValuePassword, "password"},
		{runtimecompose.ValueDatabase, "database"},
	}
	for _, field := range fields {
		value, ok := service.Values[field.key].(string)
		if !ok || value == "" {
			return fmt.Errorf("postgresql single validate %s: must be a non-empty string", field.name)
		}
	}

	return nil
}

func (*singleSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValuePort:     defaultPort,
		runtimecompose.ValueVersion:  defaultVersion,
		runtimecompose.ValueImage:    "",
		runtimecompose.ValueDataDir:  "",
		runtimecompose.ValueUser:     "postgres",
		runtimecompose.ValuePassword: "postgres123",
		runtimecompose.ValueDatabase: "app",
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
	case "v16":
		return "postgres:16"
	case "v17":
		return "postgres:17"
	default:
		return "postgres:17"
	}
}

func validateVersion(version string) error {
	switch version {
	case "v16", "v17":
		return nil
	default:
		return fmt.Errorf("unsupported version %q", version)
	}
}

func serviceEnvKeyPrefix(name string) string {
	normalized := strings.ToUpper(name)
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return "POSTGRES_" + normalized
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
