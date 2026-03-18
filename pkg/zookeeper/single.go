package zookeeper

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
	middlewareName       = "zookeeper"
	singleTemplate       = "single"
	defaultVersion       = "v3.9"
	defaultClientPort    = 2181
	defaultFollowerPort  = 2888
	defaultElectionPort  = 3888
	defaultVolumeTarget  = "/data"
	defaultDatalogTarget = "/datalog"
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
		return model.BlueprintService{}, fmt.Errorf("normalize zookeeper single version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueDataDir] = defaultStringValue(values[runtimecompose.ValueDataDir], fmt.Sprintf("./data/%s", name))
	values[runtimecompose.ValueDatalogDir] = defaultStringValue(values[runtimecompose.ValueDatalogDir], fmt.Sprintf("./datalog/%s", name))

	clientPort, err := allocateOrReservePort(values[runtimecompose.ValueClientPort], hasValue(input.Values, runtimecompose.ValueClientPort), defaultClientPort, "zookeeper single client_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueClientPort] = clientPort

	followerPort, err := allocateOrReservePort(values[runtimecompose.ValueFollowerPort], hasValue(input.Values, runtimecompose.ValueFollowerPort), defaultFollowerPort, "zookeeper single follower_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueFollowerPort] = followerPort

	electionPort, err := allocateOrReservePort(values[runtimecompose.ValueElectionPort], hasValue(input.Values, runtimecompose.ValueElectionPort), defaultElectionPort, "zookeeper single election_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueElectionPort] = electionPort

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
		datalogDir := service.Values[runtimecompose.ValueDatalogDir].(string)
		clientPort := service.Values[runtimecompose.ValueClientPort].(int)
		followerPort := service.Values[runtimecompose.ValueFollowerPort].(int)
		electionPort := service.Values[runtimecompose.ValueElectionPort].(int)
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
					"ZOO_MY_ID":                  "1",
					"ZOO_4LW_COMMANDS_WHITELIST": "ruok,mntr,srvr,stat,conf,isro",
				},
				Ports: []runtime.PortBinding{
					{HostPort: clientPort, ContainerPort: 2181, Protocol: "tcp"},
					{HostPort: followerPort, ContainerPort: 2888, Protocol: "tcp"},
					{HostPort: electionPort, ContainerPort: 3888, Protocol: "tcp"},
				},
				Volumes: []runtime.VolumeMount{
					{Source: dataDir, Target: defaultVolumeTarget},
					{Source: datalogDir, Target: defaultDatalogTarget},
				},
				HealthCheck: &runtime.HealthCheck{
					Test:        []string{"CMD-SHELL", "echo ruok | nc 127.0.0.1 2181 | grep -q imok"},
					Interval:    "5s",
					Timeout:     "5s",
					Retries:     60,
					StartPeriod: "20s",
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "zookeeper-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"%s_VERSION=%s\n%s_CLIENT_PORT=%d\n%s_FOLLOWER_PORT=%d\n%s_ELECTION_PORT=%d\n",
						envKeyPrefix, version,
						envKeyPrefix, clientPort,
						envKeyPrefix, followerPort,
						envKeyPrefix, electionPort,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "zookeeper-build",
					PathKey:   "build_script",
					Content:   fmt.Sprintf("echo \"ZooKeeper %s (%s) compose stack started\"\n", service.Name, version),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:    "zookeeper-check",
					PathKey: "check_script",
					Content: fmt.Sprintf(
						"R=\"$({ echo ruok | \"$CONTAINER_ENGINE\" exec -i %s /bin/bash -lc 'cat | nc 127.0.0.1 2181'; } | tr -d '\\r')\"\n"+
							"[ \"$R\" = \"imok\" ]\n"+
							"\"$CONTAINER_ENGINE\" exec %s /bin/bash -lc \"echo mntr | nc 127.0.0.1 2181 | grep -E 'zk_server_state|zk_version'\"\n"+
							"\"$CONTAINER_ENGINE\" exec %s /bin/bash -lc \"zkCli.sh -server 127.0.0.1:2181 create /zygarde_smoke ok >/tmp/zk.out 2>&1 || true\"\n"+
							"OUT=\"$(\"$CONTAINER_ENGINE\" exec %s /bin/bash -lc \"zkCli.sh -server 127.0.0.1:2181 get /zygarde_smoke 2>/dev/null | grep -E '^ok$' | head -n1\" | tr -d '\\r')\"\n"+
							"[ \"$OUT\" = \"ok\" ]\n",
						containerName,
						containerName,
						containerName,
						containerName,
					),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "zookeeper-readme",
					PathKey:   "readme_file",
					Content:   fmt.Sprintf("# ZooKeeper %s\n\n- version: %s\n- image: %s\n- client port: %d\n- follower port: %d\n- election port: %d\n", service.Name, version, image, clientPort, followerPort, electionPort),
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
		return fmt.Errorf("zookeeper single validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != singleTemplate {
		return fmt.Errorf("zookeeper single validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || strings.TrimSpace(version) == "" {
		return fmt.Errorf("zookeeper single validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("zookeeper single validate version: %w", err)
	}

	seen := map[int]string{}
	for _, field := range []struct {
		key  string
		name string
	}{
		{runtimecompose.ValueClientPort, "client_port"},
		{runtimecompose.ValueFollowerPort, "follower_port"},
		{runtimecompose.ValueElectionPort, "election_port"},
	} {
		port, err := normalizePort(service.Values[field.key])
		if err != nil {
			return fmt.Errorf("zookeeper single validate %s: %w", field.name, err)
		}
		if port <= 0 {
			return fmt.Errorf("zookeeper single validate %s: must be greater than 0", field.name)
		}
		if previous, ok := seen[port]; ok {
			return fmt.Errorf("zookeeper single validate ports: %s and %s must be different", previous, field.name)
		}
		seen[port] = field.name
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValueServiceName, "service_name"},
		{runtimecompose.ValueContainerName, "container_name"},
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueDataDir, "data_dir"},
		{runtimecompose.ValueDatalogDir, "datalog_dir"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("zookeeper single validate %s: must be a non-empty string", field.name)
		}
	}
	return nil
}

func (*singleSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:      defaultVersion,
		runtimecompose.ValueImage:        "",
		runtimecompose.ValueDataDir:      "",
		runtimecompose.ValueDatalogDir:   "",
		runtimecompose.ValueClientPort:   defaultClientPort,
		runtimecompose.ValueFollowerPort: defaultFollowerPort,
		runtimecompose.ValueElectionPort: defaultElectionPort,
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
	case "v3.8", "v3.9":
		return nil
	default:
		return fmt.Errorf("unsupported version %q", version)
	}
}

func imageForVersion(version string) string {
	switch version {
	case "v3.8":
		return "zookeeper:3.8"
	case "v3.9":
		return "zookeeper:3.9"
	default:
		return "zookeeper:3.9"
	}
}

func serviceEnvKeyPrefix(name string) string {
	normalized := strings.ToUpper(name)
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return "ZOOKEEPER_" + normalized
}
