package etcd

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
	middlewareName      = "etcd"
	singleTemplate      = "single"
	defaultVersion      = "v3.6"
	defaultClientPort   = 2379
	defaultPeerPort     = 2380
	defaultClusterToken = "zygarde-etcd-single"
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
		return model.BlueprintService{}, fmt.Errorf("normalize etcd single version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueDataDir] = defaultStringValue(values[runtimecompose.ValueDataDir], fmt.Sprintf("./data/%s", name))
	values[runtimecompose.ValueClusterToken] = defaultStringValue(values[runtimecompose.ValueClusterToken], defaultClusterToken)

	clientPort, err := allocateOrReservePort(values[runtimecompose.ValueClientPort], hasValue(input.Values, runtimecompose.ValueClientPort), defaultClientPort, "etcd single client_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueClientPort] = clientPort

	peerPort, err := allocateOrReservePort(values[runtimecompose.ValuePeerPort], hasValue(input.Values, runtimecompose.ValuePeerPort), defaultPeerPort, "etcd single peer_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValuePeerPort] = peerPort

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
		serviceName := service.Values[runtimecompose.ValueServiceName].(string)
		containerName := service.Values[runtimecompose.ValueContainerName].(string)
		image := service.Values[runtimecompose.ValueImage].(string)
		dataDir := service.Values[runtimecompose.ValueDataDir].(string)
		clientPort := service.Values[runtimecompose.ValueClientPort].(int)
		peerPort := service.Values[runtimecompose.ValuePeerPort].(int)
		clusterToken := service.Values[runtimecompose.ValueClusterToken].(string)
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
					"ALLOW_NONE_AUTHENTICATION":        "yes",
					"ETCD_NAME":                        serviceName,
					"ETCD_DATA_DIR":                    "/etcd-data",
					"ETCD_LISTEN_CLIENT_URLS":          "http://0.0.0.0:2379",
					"ETCD_ADVERTISE_CLIENT_URLS":       fmt.Sprintf("http://%s:2379", serviceName),
					"ETCD_LISTEN_PEER_URLS":            "http://0.0.0.0:2380",
					"ETCD_INITIAL_ADVERTISE_PEER_URLS": fmt.Sprintf("http://%s:2380", serviceName),
					"ETCD_INITIAL_CLUSTER":             fmt.Sprintf("%s=http://%s:2380", serviceName, serviceName),
					"ETCD_INITIAL_CLUSTER_STATE":       "new",
					"ETCD_INITIAL_CLUSTER_TOKEN":       clusterToken,
				},
				Ports: []runtime.PortBinding{
					{HostPort: clientPort, ContainerPort: 2379, Protocol: "tcp"},
					{HostPort: peerPort, ContainerPort: 2380, Protocol: "tcp"},
				},
				Volumes: []runtime.VolumeMount{
					{Source: dataDir, Target: "/etcd-data"},
				},
				HealthCheck: &runtime.HealthCheck{
					Test:        []string{"CMD-SHELL", "etcdctl --endpoints=http://127.0.0.1:2379 endpoint health >/dev/null 2>&1"},
					Interval:    "5s",
					Timeout:     "5s",
					Retries:     60,
					StartPeriod: "10s",
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "etcd-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"%s_VERSION=%s\n%s_IMAGE=%s\n%s_CLIENT_PORT=%d\n%s_PEER_PORT=%d\n%s_CLUSTER_TOKEN=%s\n",
						envKeyPrefix, version,
						envKeyPrefix, image,
						envKeyPrefix, clientPort,
						envKeyPrefix, peerPort,
						envKeyPrefix, clusterToken,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "etcd-build",
					PathKey:   "build_script",
					Content:   fmt.Sprintf("echo \"etcd %s (%s) compose stack started\"\n", service.Name, version),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:    "etcd-check",
					PathKey: "check_script",
					Content: fmt.Sprintf(
						"\"$CONTAINER_ENGINE\" exec %s etcdctl --endpoints=http://127.0.0.1:2379 endpoint health\n"+
							"\"$CONTAINER_ENGINE\" exec %s etcdctl --endpoints=http://127.0.0.1:2379 member list\n"+
							"KEY=\"zygarde-smoke-$(date +%%s)\"\n"+
							"VAL=\"ok-$(date +%%s)\"\n"+
							"\"$CONTAINER_ENGINE\" exec %s etcdctl --endpoints=http://127.0.0.1:2379 put \"$KEY\" \"$VAL\" >/dev/null\n"+
							"OUT=\"$(\"$CONTAINER_ENGINE\" exec %s etcdctl --endpoints=http://127.0.0.1:2379 get \"$KEY\" --print-value-only | tr -d '\\r')\"\n"+
							"[ \"$OUT\" = \"$VAL\" ]\n",
						containerName, containerName, containerName, containerName,
					),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "etcd-readme",
					PathKey:   "readme_file",
					Content:   fmt.Sprintf("# etcd %s\n\n- version: %s\n- image: %s\n- client port: %d\n- peer port: %d\n- cluster token: %s\n", service.Name, version, image, clientPort, peerPort, clusterToken),
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
		return fmt.Errorf("etcd single validate: unexpected template %q", service.Template)
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("etcd single validate: unexpected middleware %q", service.Middleware)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("etcd single validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("etcd single validate version: %w", err)
	}

	clientPort, err := normalizePort(service.Values[runtimecompose.ValueClientPort])
	if err != nil {
		return fmt.Errorf("etcd single validate client_port: %w", err)
	}
	peerPort, err := normalizePort(service.Values[runtimecompose.ValuePeerPort])
	if err != nil {
		return fmt.Errorf("etcd single validate peer_port: %w", err)
	}
	if clientPort <= 0 || peerPort <= 0 {
		return fmt.Errorf("etcd single validate ports: must be greater than 0")
	}
	if clientPort == peerPort {
		return fmt.Errorf("etcd single validate ports: client_port and peer_port must be different")
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValueServiceName, "service_name"},
		{runtimecompose.ValueContainerName, "container_name"},
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueDataDir, "data_dir"},
		{runtimecompose.ValueClusterToken, "cluster_token"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("etcd single validate %s: must be a non-empty string", field.name)
		}
	}
	return nil
}

func (*singleSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:      defaultVersion,
		runtimecompose.ValueImage:        "",
		runtimecompose.ValueDataDir:      "",
		runtimecompose.ValueClientPort:   defaultClientPort,
		runtimecompose.ValuePeerPort:     defaultPeerPort,
		runtimecompose.ValueClusterToken: defaultClusterToken,
	}
}

func defaultStringValue(value any, fallback string) string {
	current, ok := value.(string)
	if !ok || current == "" {
		return fallback
	}
	return fallbackIfWhitespace(current, fallback)
}

func fallbackIfWhitespace(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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
		return "quay.io/coreos/etcd:v3.6.0"
	}
	return "quay.io/coreos/etcd:v3.6.0"
}

func serviceEnvKeyPrefix(name string) string {
	normalized := strings.ToUpper(name)
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return "ETCD_" + normalized
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
