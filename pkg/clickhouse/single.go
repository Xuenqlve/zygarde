package clickhouse

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
	middlewareName   = "clickhouse"
	singleTemplate   = "single"
	defaultVersion   = "v25"
	defaultHTTPPort  = 8123
	defaultTCPPort   = 9000
	clickhouseTarget = "/var/lib/clickhouse"
)

func init() {
	if err := Register(runtime.EnvironmentTypeCompose); err != nil {
		panic(err)
	}
}

func Register(envType runtime.EnvironmentType) error {
	if err := tpl.RegisterMiddleware(tpl.NewMiddlewareRuntimeKey(middlewareName, singleTemplate, envType), NewSingleSpec()); err != nil {
		return err
	}
	return tpl.RegisterMiddleware(tpl.NewMiddlewareRuntimeKey(middlewareName, clusterTemplate, envType), NewClusterSpec())
}

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
		return model.BlueprintService{}, fmt.Errorf("normalize clickhouse single version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueDataDir] = defaultStringValue(values[runtimecompose.ValueDataDir], fmt.Sprintf("./data/%s", name))

	httpPort, err := allocateOrReservePort(values[runtimecompose.ValueHTTPPort], hasValue(input.Values, runtimecompose.ValueHTTPPort), defaultHTTPPort, "clickhouse single http_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueHTTPPort] = httpPort

	tcpPort, err := allocateOrReservePort(values[runtimecompose.ValueTCPPort], hasValue(input.Values, runtimecompose.ValueTCPPort), defaultTCPPort, "clickhouse single tcp_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueTCPPort] = tcpPort

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
		httpPort := service.Values[runtimecompose.ValueHTTPPort].(int)
		tcpPort := service.Values[runtimecompose.ValueTCPPort].(int)
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
					{HostPort: httpPort, ContainerPort: 8123, Protocol: "tcp"},
					{HostPort: tcpPort, ContainerPort: 9000, Protocol: "tcp"},
				},
				Volumes: []runtime.VolumeMount{
					{Source: dataDir, Target: clickhouseTarget},
				},
				HealthCheck: &runtime.HealthCheck{
					Test:        []string{"CMD-SHELL", "clickhouse-client -q \"SELECT 1\" >/dev/null 2>&1"},
					Interval:    "5s",
					Timeout:     "5s",
					Retries:     60,
					StartPeriod: "20s",
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "clickhouse-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"%s_VERSION=%s\n%s_IMAGE=%s\n%s_HTTP_PORT=%d\n%s_TCP_PORT=%d\n",
						envKeyPrefix, version,
						envKeyPrefix, image,
						envKeyPrefix, httpPort,
						envKeyPrefix, tcpPort,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "clickhouse-build",
					PathKey:   "build_script",
					Content:   fmt.Sprintf("echo \"ClickHouse %s (%s) compose stack started\"\n", service.Name, version),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:    "clickhouse-check",
					PathKey: "check_script",
					Content: fmt.Sprintf(
						"\"$CONTAINER_ENGINE\" exec %s clickhouse-client -q \"SELECT 1\"\n"+
							"\"$CONTAINER_ENGINE\" exec %s clickhouse-client -q \"SELECT version()\"\n"+
							"TABLE=\"zygarde_smoke_$(date +%%s)\"\n"+
							"\"$CONTAINER_ENGINE\" exec %s clickhouse-client -q \"CREATE TABLE ${TABLE} (id UInt32, v String) ENGINE=MergeTree ORDER BY id\"\n"+
							"\"$CONTAINER_ENGINE\" exec %s clickhouse-client -q \"INSERT INTO ${TABLE} VALUES (1, 'ok')\"\n"+
							"OUT=\"$(\"$CONTAINER_ENGINE\" exec %s clickhouse-client -q \"SELECT v FROM ${TABLE} WHERE id=1 FORMAT TSVRaw\" | tr -d '\\r')\"\n"+
							"[ \"$OUT\" = \"ok\" ]\n"+
							"\"$CONTAINER_ENGINE\" exec %s clickhouse-client -q \"DROP TABLE IF EXISTS ${TABLE}\"\n",
						containerName,
						containerName,
						containerName,
						containerName,
						containerName,
						containerName,
					),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "clickhouse-readme",
					PathKey:   "readme_file",
					Content:   fmt.Sprintf("# ClickHouse %s\n\n- version: %s\n- image: %s\n- http port: %d\n- tcp port: %d\n", service.Name, version, image, httpPort, tcpPort),
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
		return fmt.Errorf("clickhouse single validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != singleTemplate {
		return fmt.Errorf("clickhouse single validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || strings.TrimSpace(version) == "" {
		return fmt.Errorf("clickhouse single validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("clickhouse single validate version: %w", err)
	}

	httpPort, err := normalizePort(service.Values[runtimecompose.ValueHTTPPort])
	if err != nil {
		return fmt.Errorf("clickhouse single validate http_port: %w", err)
	}
	tcpPort, err := normalizePort(service.Values[runtimecompose.ValueTCPPort])
	if err != nil {
		return fmt.Errorf("clickhouse single validate tcp_port: %w", err)
	}
	if httpPort <= 0 || tcpPort <= 0 {
		return fmt.Errorf("clickhouse single validate ports: must be greater than 0")
	}
	if httpPort == tcpPort {
		return fmt.Errorf("clickhouse single validate ports: http_port and tcp_port must be different")
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValueServiceName, "service_name"},
		{runtimecompose.ValueContainerName, "container_name"},
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueDataDir, "data_dir"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("clickhouse single validate %s: must be a non-empty string", field.name)
		}
	}
	return nil
}

func (*singleSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:  defaultVersion,
		runtimecompose.ValueImage:    "",
		runtimecompose.ValueDataDir:  "",
		runtimecompose.ValueHTTPPort: defaultHTTPPort,
		runtimecompose.ValueTCPPort:  defaultTCPPort,
	}
}

func defaultStringValue(value any, fallback string) string {
	current, ok := value.(string)
	if !ok || strings.TrimSpace(current) == "" {
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

func validateVersion(version string) error {
	switch version {
	case "v24", "v25":
		return nil
	default:
		return fmt.Errorf("unsupported version %q", version)
	}
}

func imageForVersion(version string) string {
	switch version {
	case "v24":
		return "clickhouse/clickhouse-server:24"
	case "v25":
		return "clickhouse/clickhouse-server:25.8"
	default:
		return "clickhouse/clickhouse-server:25.8"
	}
}

func serviceEnvKeyPrefix(name string) string {
	normalized := strings.ToUpper(name)
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return "CLICKHOUSE_" + normalized
}
