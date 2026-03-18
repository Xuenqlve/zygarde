package consul

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
	middlewareName    = "consul"
	singleTemplate    = "single"
	defaultVersion    = "v1.20"
	defaultHTTPPort   = 8500
	defaultDNSPort    = 8600
	defaultServerPort = 8300
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
		return model.BlueprintService{}, fmt.Errorf("normalize consul single version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueDataDir] = defaultStringValue(values[runtimecompose.ValueDataDir], fmt.Sprintf("./data/%s", name))

	httpPort, err := allocateOrReservePort(values[runtimecompose.ValueHTTPPort], hasValue(input.Values, runtimecompose.ValueHTTPPort), defaultHTTPPort, "consul single http_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueHTTPPort] = httpPort
	dnsPort, err := allocateOrReservePort(values[runtimecompose.ValueDNSPort], hasValue(input.Values, runtimecompose.ValueDNSPort), defaultDNSPort, "consul single dns_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueDNSPort] = dnsPort
	serverPort, err := allocateOrReservePort(values[runtimecompose.ValueServerPort], hasValue(input.Values, runtimecompose.ValueServerPort), defaultServerPort, "consul single server_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueServerPort] = serverPort

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
		containerName := service.Values[runtimecompose.ValueContainerName].(string)
		image := service.Values[runtimecompose.ValueImage].(string)
		dataDir := service.Values[runtimecompose.ValueDataDir].(string)
		httpPort := service.Values[runtimecompose.ValueHTTPPort].(int)
		dnsPort := service.Values[runtimecompose.ValueDNSPort].(int)
		serverPort := service.Values[runtimecompose.ValueServerPort].(int)
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
				Command: []string{
					"agent",
					"-server",
					"-ui",
					"-node=consul1",
					"-bootstrap-expect=1",
					"-client=0.0.0.0",
					"-bind=0.0.0.0",
					"-data-dir=/consul/data",
				},
				Ports: []runtime.PortBinding{
					{HostPort: httpPort, ContainerPort: 8500, Protocol: "tcp"},
					{HostPort: dnsPort, ContainerPort: 8600, Protocol: "udp"},
					{HostPort: serverPort, ContainerPort: 8300, Protocol: "tcp"},
				},
				Volumes: []runtime.VolumeMount{
					{Source: dataDir, Target: "/consul/data"},
				},
				HealthCheck: &runtime.HealthCheck{
					Test:        []string{"CMD-SHELL", "consul info >/dev/null 2>&1"},
					Interval:    "5s",
					Timeout:     "5s",
					Retries:     60,
					StartPeriod: "10s",
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "consul-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"%s_VERSION=%s\n%s_IMAGE=%s\n%s_HTTP_PORT=%d\n%s_DNS_PORT=%d\n%s_SERVER_PORT=%d\n",
						envKeyPrefix, version,
						envKeyPrefix, image,
						envKeyPrefix, httpPort,
						envKeyPrefix, dnsPort,
						envKeyPrefix, serverPort,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "consul-build",
					PathKey:   "build_script",
					Content:   fmt.Sprintf("echo \"consul %s (%s) compose stack started\"\n", service.Name, version),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:    "consul-check",
					PathKey: "check_script",
					Content: fmt.Sprintf(
						"LEADER=\"\"\nfor _ in $(seq 1 30); do\n    LEADER=\"$(curl -fsS \"http://127.0.0.1:%d/v1/status/leader\" 2>/dev/null | tr -d '\\\"' || true)\"\n    if [ -n \"$LEADER\" ]; then\n        break\n    fi\n    sleep 1\ndone\n[ -n \"$LEADER\" ]\n"+
							"MEMBERS_JSON=\"$(curl -fsS \"http://127.0.0.1:%d/v1/agent/members\")\"\n"+
							"echo \"$MEMBERS_JSON\" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(\"member_count=\",len(d));'\n"+
							"KEY=\"zygarde/smoke/$(date +%%s)\"\nVAL=\"ok-$(date +%%s)\"\n"+
							"curl -fsS -X PUT --data \"$VAL\" \"http://127.0.0.1:%d/v1/kv/$KEY\" >/dev/null\n"+
							"OUT=\"$(curl -fsS \"http://127.0.0.1:%d/v1/kv/$KEY?raw\")\"\n[ \"$OUT\" = \"$VAL\" ]\n",
						httpPort, httpPort, httpPort, httpPort,
					),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "consul-readme",
					PathKey:   "readme_file",
					Content:   fmt.Sprintf("# Consul %s\n\n- version: %s\n- image: %s\n- http port: %d\n- dns port: %d\n- server port: %d\n", service.Name, version, image, httpPort, dnsPort, serverPort),
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
	if service.Template != singleTemplate {
		return fmt.Errorf("consul single validate: unexpected template %q", service.Template)
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("consul single validate: unexpected middleware %q", service.Middleware)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("consul single validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("consul single validate version: %w", err)
	}

	httpPort, err := normalizePort(service.Values[runtimecompose.ValueHTTPPort])
	if err != nil {
		return fmt.Errorf("consul single validate http_port: %w", err)
	}
	dnsPort, err := normalizePort(service.Values[runtimecompose.ValueDNSPort])
	if err != nil {
		return fmt.Errorf("consul single validate dns_port: %w", err)
	}
	serverPort, err := normalizePort(service.Values[runtimecompose.ValueServerPort])
	if err != nil {
		return fmt.Errorf("consul single validate server_port: %w", err)
	}
	if httpPort <= 0 || dnsPort <= 0 || serverPort <= 0 {
		return fmt.Errorf("consul single validate ports: must be greater than 0")
	}
	seen := map[int]string{}
	for port, name := range map[int]string{httpPort: "http_port", dnsPort: "dns_port", serverPort: "server_port"} {
		if previous, ok := seen[port]; ok {
			return fmt.Errorf("consul single validate ports: %s and %s must be different", previous, name)
		}
		seen[port] = name
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValueServiceName, "service_name"},
		{runtimecompose.ValueContainerName, "container_name"},
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueDataDir, "data_dir"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("consul single validate %s: must be a non-empty string", field.name)
		}
	}
	return nil
}

func (*singleSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:    defaultVersion,
		runtimecompose.ValueImage:      "",
		runtimecompose.ValueDataDir:    "",
		runtimecompose.ValueHTTPPort:   defaultHTTPPort,
		runtimecompose.ValueDNSPort:    defaultDNSPort,
		runtimecompose.ValueServerPort: defaultServerPort,
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

func validateVersion(version string) error {
	if version != defaultVersion {
		return fmt.Errorf("unsupported version %q", version)
	}
	return nil
}

func imageForVersion(version string) string {
	if version == defaultVersion {
		return "hashicorp/consul:1.20"
	}
	return "hashicorp/consul:1.20"
}

func serviceEnvKeyPrefix(name string) string {
	normalized := strings.ToUpper(name)
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return "CONSUL_" + normalized
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
