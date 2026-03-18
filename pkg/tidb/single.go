package tidb

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
	middlewareName        = "tidb"
	singleTemplate        = "single"
	defaultVersion        = "v6.7"
	defaultPDPort         = 2379
	defaultTiKVPort       = 20160
	defaultTiDBPort       = 4000
	defaultTiDBStatusPort = 10080
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

	version := defaultStringValue(values[runtimecompose.ValueVersion], defaultVersion)
	if err := validateVersion(version); err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize tidb single version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version

	values[runtimecompose.ValuePDServiceName] = defaultStringValue(values[runtimecompose.ValuePDServiceName], name+"-pd")
	values[runtimecompose.ValueTiKVServiceName] = defaultStringValue(values[runtimecompose.ValueTiKVServiceName], name+"-tikv")
	values[runtimecompose.ValueTiDBServiceName] = defaultStringValue(values[runtimecompose.ValueTiDBServiceName], name+"-tidb")
	values[runtimecompose.ValuePDContainerName] = defaultStringValue(values[runtimecompose.ValuePDContainerName], values[runtimecompose.ValuePDServiceName].(string))
	values[runtimecompose.ValueTiKVContainerName] = defaultStringValue(values[runtimecompose.ValueTiKVContainerName], values[runtimecompose.ValueTiKVServiceName].(string))
	values[runtimecompose.ValueTiDBContainerName] = defaultStringValue(values[runtimecompose.ValueTiDBContainerName], values[runtimecompose.ValueTiDBServiceName].(string))
	values[runtimecompose.ValuePDImage] = defaultStringValue(values[runtimecompose.ValuePDImage], pdImageForVersion(version))
	values[runtimecompose.ValueTiKVImage] = defaultStringValue(values[runtimecompose.ValueTiKVImage], tikvImageForVersion(version))
	values[runtimecompose.ValueTiDBImage] = defaultStringValue(values[runtimecompose.ValueTiDBImage], tidbImageForVersion(version))
	values[runtimecompose.ValuePDDataDir] = defaultStringValue(values[runtimecompose.ValuePDDataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValuePDServiceName].(string)))
	values[runtimecompose.ValueTiKVDataDir] = defaultStringValue(values[runtimecompose.ValueTiKVDataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueTiKVServiceName].(string)))

	var err error
	values[runtimecompose.ValuePDPort], err = allocateOrReservePort(values[runtimecompose.ValuePDPort], hasValue(input.Values, runtimecompose.ValuePDPort), defaultPDPort, "tidb single pd_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueTiKVPort], err = allocateOrReservePort(values[runtimecompose.ValueTiKVPort], hasValue(input.Values, runtimecompose.ValueTiKVPort), defaultTiKVPort, "tidb single tikv_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueTiDBPort], err = allocateOrReservePort(values[runtimecompose.ValueTiDBPort], hasValue(input.Values, runtimecompose.ValueTiDBPort), defaultTiDBPort, "tidb single tidb_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueTiDBStatusPort], err = allocateOrReservePort(values[runtimecompose.ValueTiDBStatusPort], hasValue(input.Values, runtimecompose.ValueTiDBStatusPort), defaultTiDBStatusPort, "tidb single tidb_status_port")
	if err != nil {
		return model.BlueprintService{}, err
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
	services := append([]model.BlueprintService(nil), s.services...)
	s.services = nil

	contexts := make([]runtime.EnvironmentContext, 0, len(services)*3)
	for _, service := range services {
		if err := s.Validate(service); err != nil {
			return nil, err
		}

		version := service.Values[runtimecompose.ValueVersion].(string)
		pdServiceName := service.Values[runtimecompose.ValuePDServiceName].(string)
		tikvServiceName := service.Values[runtimecompose.ValueTiKVServiceName].(string)
		tidbServiceName := service.Values[runtimecompose.ValueTiDBServiceName].(string)
		pdContainerName := service.Values[runtimecompose.ValuePDContainerName].(string)
		tikvContainerName := service.Values[runtimecompose.ValueTiKVContainerName].(string)
		tidbContainerName := service.Values[runtimecompose.ValueTiDBContainerName].(string)
		pdImage := service.Values[runtimecompose.ValuePDImage].(string)
		tikvImage := service.Values[runtimecompose.ValueTiKVImage].(string)
		tidbImage := service.Values[runtimecompose.ValueTiDBImage].(string)
		pdDataDir := service.Values[runtimecompose.ValuePDDataDir].(string)
		tikvDataDir := service.Values[runtimecompose.ValueTiKVDataDir].(string)
		pdPort := service.Values[runtimecompose.ValuePDPort].(int)
		tikvPort := service.Values[runtimecompose.ValueTiKVPort].(int)
		tidbPort := service.Values[runtimecompose.ValueTiDBPort].(int)
		tidbStatusPort := service.Values[runtimecompose.ValueTiDBStatusPort].(int)

		pdContext := runtime.ComposeContext{
			EnvType:     runtimeType,
			ServiceName: pdServiceName,
			Middleware:  service.Middleware,
			Template:    service.Template,
			Service: runtime.ServiceSpec{
				Image:         pdImage,
				ContainerName: pdContainerName,
				Restart:       "unless-stopped",
				Command: []string{
					"--name=pd",
					"--data-dir=/data/pd",
					"--client-urls=http://0.0.0.0:2379",
					"--peer-urls=http://0.0.0.0:2380",
					fmt.Sprintf("--advertise-client-urls=http://%s:2379", pdServiceName),
					fmt.Sprintf("--advertise-peer-urls=http://%s:2380", pdServiceName),
					fmt.Sprintf("--initial-cluster=pd=http://%s:2380", pdServiceName),
				},
				Ports: []runtime.PortBinding{
					{HostPort: pdPort, ContainerPort: 2379, Protocol: "tcp"},
				},
				Volumes: []runtime.VolumeMount{
					{Source: pdDataDir, Target: "/data/pd"},
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "tidb-pd-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"TIDB_%s_VERSION=%s\nTIDB_%s_IMAGE=%s\nTIDB_%s_PORT=%d\n",
						serviceEnvKeySuffix(pdServiceName), version,
						serviceEnvKeySuffix(pdServiceName), pdImage,
						serviceEnvKeySuffix(pdServiceName), pdPort,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
			},
			Metadata: tpl.MergeValues(nil, service.Values),
		}

		tikvContext := runtime.ComposeContext{
			EnvType:     runtimeType,
			ServiceName: tikvServiceName,
			Middleware:  service.Middleware,
			Template:    service.Template,
			Service: runtime.ServiceSpec{
				Image:         tikvImage,
				ContainerName: tikvContainerName,
				Restart:       "unless-stopped",
				Command: []string{
					fmt.Sprintf("--pd=%s:2379", pdServiceName),
					"--addr=0.0.0.0:20160",
					fmt.Sprintf("--advertise-addr=%s:20160", tikvServiceName),
					"--data-dir=/data/tikv",
				},
				Ports: []runtime.PortBinding{
					{HostPort: tikvPort, ContainerPort: 20160, Protocol: "tcp"},
				},
				Volumes: []runtime.VolumeMount{
					{Source: tikvDataDir, Target: "/data/tikv"},
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "tidb-tikv-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"TIDB_%s_IMAGE=%s\nTIDB_%s_PORT=%d\n",
						serviceEnvKeySuffix(tikvServiceName), tikvImage,
						serviceEnvKeySuffix(tikvServiceName), tikvPort,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
			},
			Metadata: tpl.MergeValues(nil, service.Values),
		}

		tidbContext := runtime.ComposeContext{
			EnvType:     runtimeType,
			ServiceName: tidbServiceName,
			Middleware:  service.Middleware,
			Template:    service.Template,
			Service: runtime.ServiceSpec{
				Image:         tidbImage,
				ContainerName: tidbContainerName,
				Restart:       "unless-stopped",
				Command: []string{
					"--store=tikv",
					fmt.Sprintf("--path=%s:2379", pdServiceName),
					"--host=0.0.0.0",
					"--status=10080",
					fmt.Sprintf("--advertise-address=%s", tidbServiceName),
				},
				Ports: []runtime.PortBinding{
					{HostPort: tidbPort, ContainerPort: 4000, Protocol: "tcp"},
					{HostPort: tidbStatusPort, ContainerPort: 10080, Protocol: "tcp"},
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "tidb-tidb-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"TIDB_%s_IMAGE=%s\nTIDB_%s_PORT=%d\nTIDB_%s_STATUS_PORT=%d\n",
						serviceEnvKeySuffix(tidbServiceName), tidbImage,
						serviceEnvKeySuffix(tidbServiceName), tidbPort,
						serviceEnvKeySuffix(tidbServiceName), tidbStatusPort,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "tidb-build",
					PathKey:   "build_script",
					Content:   fmt.Sprintf("echo \"TiDB %s (%s) compose stack started\"\n", service.Name, version),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:    "tidb-check",
					PathKey: "check_script",
					Content: fmt.Sprintf(
						"curl -fsS \"http://127.0.0.1:%d/status\"\n"+
							"echo\n"+
							"curl -fsS \"http://127.0.0.1:%d/pd/api/v1/health\"\n"+
							"echo\n"+
							"if (exec 3<>/dev/tcp/127.0.0.1/%d) 2>/dev/null; then\n    echo \"tidb sql port %d is reachable\"\n    exec 3>&-\nelse\n    echo \"tidb sql port %d is not reachable\" >&2\n    exit 1\nfi\n",
						tidbStatusPort, pdPort, tidbPort, tidbPort, tidbPort,
					),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "tidb-readme",
					PathKey:   "readme_file",
					Content:   fmt.Sprintf("# TiDB %s\n\n- version: %s\n- pd image: %s\n- tikv image: %s\n- tidb image: %s\n- pd port: %d\n- tikv port: %d\n- tidb port: %d\n- tidb status port: %d\n", service.Name, version, pdImage, tikvImage, tidbImage, pdPort, tikvPort, tidbPort, tidbStatusPort),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeReadme,
				},
			},
			Metadata: tpl.MergeValues(nil, service.Values),
		}

		contexts = append(contexts, pdContext, tikvContext, tidbContext)
	}
	return contexts, nil
}

func (*singleSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template != singleTemplate {
		return fmt.Errorf("tidb single validate: unexpected template %q", service.Template)
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("tidb single validate: unexpected middleware %q", service.Middleware)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("tidb single validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("tidb single validate version: %w", err)
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValuePDServiceName, "pd_service_name"},
		{runtimecompose.ValueTiKVServiceName, "tikv_service_name"},
		{runtimecompose.ValueTiDBServiceName, "tidb_service_name"},
		{runtimecompose.ValuePDContainerName, "pd_container_name"},
		{runtimecompose.ValueTiKVContainerName, "tikv_container_name"},
		{runtimecompose.ValueTiDBContainerName, "tidb_container_name"},
		{runtimecompose.ValuePDImage, "pd_image"},
		{runtimecompose.ValueTiKVImage, "tikv_image"},
		{runtimecompose.ValueTiDBImage, "tidb_image"},
		{runtimecompose.ValuePDDataDir, "pd_data_dir"},
		{runtimecompose.ValueTiKVDataDir, "tikv_data_dir"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("tidb single validate %s: must be a non-empty string", field.name)
		}
	}

	portKeys := []struct{ key, name string }{
		{runtimecompose.ValuePDPort, "pd_port"},
		{runtimecompose.ValueTiKVPort, "tikv_port"},
		{runtimecompose.ValueTiDBPort, "tidb_port"},
		{runtimecompose.ValueTiDBStatusPort, "tidb_status_port"},
	}
	seen := make(map[int]string, len(portKeys))
	for _, field := range portKeys {
		port, err := normalizePort(service.Values[field.key])
		if err != nil {
			return fmt.Errorf("tidb single validate %s: %w", field.name, err)
		}
		if port <= 0 {
			return fmt.Errorf("tidb single validate %s: must be greater than 0", field.name)
		}
		if previous, ok := seen[port]; ok {
			return fmt.Errorf("tidb single validate ports: %s and %s must be different", previous, field.name)
		}
		seen[port] = field.name
	}
	return nil
}

func (*singleSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:           defaultVersion,
		runtimecompose.ValuePDServiceName:     "",
		runtimecompose.ValueTiKVServiceName:   "",
		runtimecompose.ValueTiDBServiceName:   "",
		runtimecompose.ValuePDContainerName:   "",
		runtimecompose.ValueTiKVContainerName: "",
		runtimecompose.ValueTiDBContainerName: "",
		runtimecompose.ValuePDImage:           "",
		runtimecompose.ValueTiKVImage:         "",
		runtimecompose.ValueTiDBImage:         "",
		runtimecompose.ValuePDDataDir:         "",
		runtimecompose.ValueTiKVDataDir:       "",
		runtimecompose.ValuePDPort:            defaultPDPort,
		runtimecompose.ValueTiKVPort:          defaultTiKVPort,
		runtimecompose.ValueTiDBPort:          defaultTiDBPort,
		runtimecompose.ValueTiDBStatusPort:    defaultTiDBStatusPort,
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

func validateVersion(version string) error {
	if version != defaultVersion {
		return fmt.Errorf("unsupported version %q", version)
	}
	return nil
}

func pdImageForVersion(version string) string {
	if version == defaultVersion {
		return "pingcap/pd:v6.5.12"
	}
	return "pingcap/pd:v6.5.12"
}

func tikvImageForVersion(version string) string {
	if version == defaultVersion {
		return "pingcap/tikv:v6.5.12"
	}
	return "pingcap/tikv:v6.5.12"
}

func tidbImageForVersion(version string) string {
	if version == defaultVersion {
		return "pingcap/tidb:v6.5.12"
	}
	return "pingcap/tidb:v6.5.12"
}

func serviceEnvKeySuffix(name string) string {
	normalized := strings.ToUpper(name)
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return normalized
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
