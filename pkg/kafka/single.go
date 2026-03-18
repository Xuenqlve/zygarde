package kafka

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
	middlewareName   = "kafka"
	singleTemplate   = "single"
	defaultVersion   = "v4.2"
	defaultPort      = 9092
	defaultClusterID = "MkU3OEVBNTcwNTJENDM2Qk"
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
		return model.BlueprintService{}, fmt.Errorf("normalize kafka single version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueDataDir] = defaultStringValue(values[runtimecompose.ValueDataDir], fmt.Sprintf("./data/%s", name))
	values[runtimecompose.ValueClusterID] = defaultStringValue(values[runtimecompose.ValueClusterID], defaultClusterID)

	port, err := allocateOrReservePort(values[runtimecompose.ValuePort], hasValue(input.Values, runtimecompose.ValuePort), defaultPort, "kafka single port")
	if err != nil {
		return model.BlueprintService{}, err
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

		port := service.Values[runtimecompose.ValuePort].(int)
		version := service.Values[runtimecompose.ValueVersion].(string)
		image := service.Values[runtimecompose.ValueImage].(string)
		containerName := service.Values[runtimecompose.ValueContainerName].(string)
		dataDir := service.Values[runtimecompose.ValueDataDir].(string)
		clusterID := service.Values[runtimecompose.ValueClusterID].(string)
		envKeyPrefix := serviceEnvKeyPrefix(service.Name)

		contexts = append(contexts, runtime.ComposeContext{
			EnvType:     runtimeType,
			ServiceName: service.Name,
			Middleware:  service.Middleware,
			Template:    service.Template,
			Service: runtime.ServiceSpec{
				Image:         image,
				Hostname:      "kafka",
				ContainerName: containerName,
				Restart:       "unless-stopped",
				Environment: map[string]string{
					"KAFKA_NODE_ID":                        "1",
					"KAFKA_PROCESS_ROLES":                  "broker,controller",
					"KAFKA_CONTROLLER_QUORUM_VOTERS":       "1@kafka:9093",
					"KAFKA_LISTENERS":                      "PLAINTEXT://:9092,CONTROLLER://:9093",
					"KAFKA_ADVERTISED_LISTENERS":           "PLAINTEXT://kafka:9092",
					"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP": "CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT",
					"KAFKA_CONTROLLER_LISTENER_NAMES":      "CONTROLLER",
					"KAFKA_INTER_BROKER_LISTENER_NAME":     "PLAINTEXT",
					"KAFKA_LOG_DIRS":                       "/var/lib/kafka/data",
					"CLUSTER_ID":                           clusterID,
				},
				Ports: []runtime.PortBinding{
					{HostPort: port, ContainerPort: 9092, Protocol: "tcp"},
				},
				Volumes: []runtime.VolumeMount{
					{Source: dataDir, Target: "/var/lib/kafka/data"},
				},
				HealthCheck: &runtime.HealthCheck{
					Test:        []string{"CMD-SHELL", "bash -lc '/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list >/dev/null 2>&1'"},
					Interval:    "5s",
					Timeout:     "5s",
					Retries:     60,
					StartPeriod: "30s",
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "kafka-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"%s_VERSION=%s\n%s_IMAGE=%s\n%s_PORT=%d\n%s_CLUSTER_ID=%s\n",
						envKeyPrefix, version,
						envKeyPrefix, image,
						envKeyPrefix, port,
						envKeyPrefix, clusterID,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "kafka-build",
					PathKey:   "build_script",
					Content:   fmt.Sprintf("echo \"Kafka %s (%s) compose stack started\"\n", service.Name, version),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:    "kafka-check",
					PathKey: "check_script",
					Content: fmt.Sprintf(
						"\"$CONTAINER_ENGINE\" exec %s bash -lc '/opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092 >/dev/null'\n"+
							"TOPIC=\"zygarde-smoke-$(date +%%s)\"\n"+
							"MSG=\"hello-zygarde-$(date +%%s)\"\n"+
							"\"$CONTAINER_ENGINE\" exec %s bash -lc \"/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic ${TOPIC} --partitions 1 --replication-factor 1 >/dev/null\"\n"+
							"echo \"$MSG\" | \"$CONTAINER_ENGINE\" exec -i %s bash -lc \"/opt/kafka/bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic ${TOPIC} >/dev/null 2>&1\"\n"+
							"ok=0\nfor _ in $(seq 1 8); do\n    OFF=\"$(\"$CONTAINER_ENGINE\" exec %s bash -lc \\\"/opt/kafka/bin/kafka-get-offsets.sh --bootstrap-server localhost:9092 --topic ${TOPIC} 2>/dev/null\\\" | awk -F: '{print $3}' | head -n1 | tr -d '[:space:]')\"\n    if [ -n \"$OFF\" ] && [ \"$OFF\" -ge 1 ] 2>/dev/null; then\n        ok=1\n        break\n    fi\n    sleep 1\ndone\n[ \"$ok\" -eq 1 ]\n"+
							"\"$CONTAINER_ENGINE\" exec %s bash -lc \"/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic ${TOPIC}\"\n",
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
					Name:      "kafka-readme",
					PathKey:   "readme_file",
					Content:   fmt.Sprintf("# Kafka %s\n\n- version: %s\n- image: %s\n- port: %d\n- cluster id: %s\n", service.Name, version, image, port, clusterID),
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
		return fmt.Errorf("kafka single validate: unexpected template %q", service.Template)
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("kafka single validate: unexpected middleware %q", service.Middleware)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("kafka single validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("kafka single validate version: %w", err)
	}

	port, err := normalizePort(service.Values[runtimecompose.ValuePort])
	if err != nil {
		return fmt.Errorf("kafka single validate port: %w", err)
	}
	if port <= 0 {
		return fmt.Errorf("kafka single validate port: must be greater than 0")
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValueServiceName, "service_name"},
		{runtimecompose.ValueContainerName, "container_name"},
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueDataDir, "data_dir"},
		{runtimecompose.ValueClusterID, "cluster_id"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("kafka single validate %s: must be a non-empty string", field.name)
		}
	}
	return nil
}

func (*singleSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:   defaultVersion,
		runtimecompose.ValueImage:     "",
		runtimecompose.ValueDataDir:   "",
		runtimecompose.ValuePort:      defaultPort,
		runtimecompose.ValueClusterID: defaultClusterID,
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
		return "apache/kafka:4.2.0"
	}
	return "apache/kafka:4.2.0"
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
	return "KAFKA_" + normalized
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
