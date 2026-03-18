package kafka

import (
	"fmt"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

const (
	clusterTemplate   = "cluster"
	defaultKafka1Port = 9092
	defaultKafka2Port = 9094
	defaultKafka3Port = 9096
)

func NewClusterSpec() tpl.Middleware {
	return &clusterSpec{}
}

type clusterSpec struct {
	services []model.BlueprintService
}

func (*clusterSpec) Middleware() string { return middlewareName }
func (*clusterSpec) Template() string   { return clusterTemplate }
func (*clusterSpec) IsDefault() bool    { return false }

func (s *clusterSpec) Normalize(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	name := input.Name
	if name == "" {
		name = tpl.DefaultServiceName(s.Middleware(), index)
	}

	values := tpl.MergeValues(s.DefaultValues(), input.Values)

	version := defaultStringValue(values[runtimecompose.ValueVersion], defaultVersion)
	if err := validateVersion(version); err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize kafka cluster version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueClusterID] = defaultStringValue(values[runtimecompose.ValueClusterID], defaultClusterID)

	values[runtimecompose.ValueKafka1ServiceName] = defaultStringValue(values[runtimecompose.ValueKafka1ServiceName], name+"-kafka1")
	values[runtimecompose.ValueKafka2ServiceName] = defaultStringValue(values[runtimecompose.ValueKafka2ServiceName], name+"-kafka2")
	values[runtimecompose.ValueKafka3ServiceName] = defaultStringValue(values[runtimecompose.ValueKafka3ServiceName], name+"-kafka3")
	values[runtimecompose.ValueKafka1ContainerName] = defaultStringValue(values[runtimecompose.ValueKafka1ContainerName], values[runtimecompose.ValueKafka1ServiceName].(string))
	values[runtimecompose.ValueKafka2ContainerName] = defaultStringValue(values[runtimecompose.ValueKafka2ContainerName], values[runtimecompose.ValueKafka2ServiceName].(string))
	values[runtimecompose.ValueKafka3ContainerName] = defaultStringValue(values[runtimecompose.ValueKafka3ContainerName], values[runtimecompose.ValueKafka3ServiceName].(string))
	values[runtimecompose.ValueKafka1DataDir] = defaultStringValue(values[runtimecompose.ValueKafka1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueKafka1ServiceName].(string)))
	values[runtimecompose.ValueKafka2DataDir] = defaultStringValue(values[runtimecompose.ValueKafka2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueKafka2ServiceName].(string)))
	values[runtimecompose.ValueKafka3DataDir] = defaultStringValue(values[runtimecompose.ValueKafka3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueKafka3ServiceName].(string)))

	var err error
	values[runtimecompose.ValueKafka1Port], err = allocateOrReservePort(values[runtimecompose.ValueKafka1Port], hasValue(input.Values, runtimecompose.ValueKafka1Port), defaultKafka1Port, "kafka cluster kafka1_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueKafka2Port], err = allocateOrReservePort(values[runtimecompose.ValueKafka2Port], hasValue(input.Values, runtimecompose.ValueKafka2Port), defaultKafka2Port, "kafka cluster kafka2_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueKafka3Port], err = allocateOrReservePort(values[runtimecompose.ValueKafka3Port], hasValue(input.Values, runtimecompose.ValueKafka3Port), defaultKafka3Port, "kafka cluster kafka3_port")
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

func (s *clusterSpec) Configure(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
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

func (s *clusterSpec) BuildRuntimeContexts(runtimeType runtime.EnvironmentType) ([]runtime.EnvironmentContext, error) {
	services := append([]model.BlueprintService(nil), s.services...)
	s.services = nil

	contexts := make([]runtime.EnvironmentContext, 0, len(services)*3)
	for _, service := range services {
		if err := s.Validate(service); err != nil {
			return nil, err
		}

		image := service.Values[runtimecompose.ValueImage].(string)
		version := service.Values[runtimecompose.ValueVersion].(string)
		clusterID := service.Values[runtimecompose.ValueClusterID].(string)

		type nodeSpec struct {
			serviceKey   string
			containerKey string
			dataDirKey   string
			portKey      string
			hostname     string
			nodeID       string
		}
		nodes := []nodeSpec{
			{runtimecompose.ValueKafka1ServiceName, runtimecompose.ValueKafka1ContainerName, runtimecompose.ValueKafka1DataDir, runtimecompose.ValueKafka1Port, "kafka1", "1"},
			{runtimecompose.ValueKafka2ServiceName, runtimecompose.ValueKafka2ContainerName, runtimecompose.ValueKafka2DataDir, runtimecompose.ValueKafka2Port, "kafka2", "2"},
			{runtimecompose.ValueKafka3ServiceName, runtimecompose.ValueKafka3ContainerName, runtimecompose.ValueKafka3DataDir, runtimecompose.ValueKafka3Port, "kafka3", "3"},
		}

		quorumVoters := "1@kafka1:9093,2@kafka2:9093,3@kafka3:9093"
		clusterContexts := make([]runtime.ComposeContext, 0, len(nodes))
		for _, node := range nodes {
			serviceName := service.Values[node.serviceKey].(string)
			containerName := service.Values[node.containerKey].(string)
			dataDir := service.Values[node.dataDirKey].(string)
			port := service.Values[node.portKey].(int)
			envKeyPrefix := serviceEnvKeyPrefix(serviceName)

			clusterContexts = append(clusterContexts, runtime.ComposeContext{
				EnvType:     runtimeType,
				ServiceName: serviceName,
				Middleware:  service.Middleware,
				Template:    service.Template,
				Service: runtime.ServiceSpec{
					Image:         image,
					Hostname:      node.hostname,
					ContainerName: containerName,
					Restart:       "unless-stopped",
					Environment: map[string]string{
						"KAFKA_NODE_ID":                        node.nodeID,
						"KAFKA_PROCESS_ROLES":                  "broker,controller",
						"KAFKA_CONTROLLER_QUORUM_VOTERS":       quorumVoters,
						"KAFKA_LISTENERS":                      "PLAINTEXT://:9092,CONTROLLER://:9093",
						"KAFKA_ADVERTISED_LISTENERS":           fmt.Sprintf("PLAINTEXT://%s:9092", node.hostname),
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
						Name:    "kafka-cluster-env",
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
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			})
		}

		container1 := service.Values[runtimecompose.ValueKafka1ContainerName].(string)
		container2 := service.Values[runtimecompose.ValueKafka2ContainerName].(string)
		clusterContexts[0] = addClusterSharedAssets(clusterContexts[0], service, version, image, container1, container2)
		for _, contextItem := range clusterContexts {
			contexts = append(contexts, contextItem)
		}
	}
	return contexts, nil
}

func addClusterSharedAssets(context runtime.ComposeContext, service model.BlueprintService, version, image, container1, container2 string) runtime.ComposeContext {
	context.Assets = append(context.Assets,
		runtime.AssetSpec{
			Name:      "kafka-cluster-build",
			PathKey:   "build_script",
			Content:   fmt.Sprintf("echo \"Kafka %s (%s) cluster compose stack started\"\n", service.Name, version),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:    "kafka-cluster-check",
			PathKey: "check_script",
			Content: fmt.Sprintf(
				"META=\"$(\"$CONTAINER_ENGINE\" exec %s bash -lc '/opt/kafka/bin/kafka-metadata-quorum.sh --bootstrap-server kafka1:9092 describe --status' 2>/dev/null || true)\"\n"+
					"echo \"$META\"\n"+
					"echo \"$META\" | grep -q 'LeaderId'\n"+
					"TOPIC=\"zygarde-smoke\"\n"+
					"\"$CONTAINER_ENGINE\" exec %s bash -lc '/opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka1:9092 --create --if-not-exists --topic zygarde-smoke --partitions 1 --replication-factor 3 >/dev/null'\n"+
					"echo 'hello-kafka' | \"$CONTAINER_ENGINE\" exec -i %s bash -lc '/opt/kafka/bin/kafka-console-producer.sh --bootstrap-server kafka1:9092 --topic zygarde-smoke >/dev/null 2>&1'\n"+
					"OUT=\"$(\"$CONTAINER_ENGINE\" exec %s bash -lc '/opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server kafka2:9092 --topic zygarde-smoke --from-beginning --max-messages 1 --timeout-ms 8000 2>/dev/null' | tr -d '\\r')\"\n"+
					"[ \"$OUT\" = \"hello-kafka\" ]\n",
				container1, container1, container1, container2,
			),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:      "kafka-cluster-readme",
			PathKey:   "readme_file",
			Content:   kafkaClusterReadme(service, version, image),
			Mode:      0o644,
			MergeMode: runtime.AssetMergeReadme,
		},
	)
	return context
}

func kafkaClusterReadme(service model.BlueprintService, version, image string) string {
	return fmt.Sprintf(
		"# Kafka %s Cluster\n\n- version: %s\n- image: %s\n- cluster id: %s\n- kafka1 port: %d\n- kafka2 port: %d\n- kafka3 port: %d\n",
		service.Name,
		version,
		image,
		service.Values[runtimecompose.ValueClusterID].(string),
		service.Values[runtimecompose.ValueKafka1Port].(int),
		service.Values[runtimecompose.ValueKafka2Port].(int),
		service.Values[runtimecompose.ValueKafka3Port].(int),
	)
}

func (*clusterSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template != clusterTemplate {
		return fmt.Errorf("kafka cluster validate: unexpected template %q", service.Template)
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("kafka cluster validate: unexpected middleware %q", service.Middleware)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("kafka cluster validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("kafka cluster validate version: %w", err)
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueClusterID, "cluster_id"},
		{runtimecompose.ValueKafka1ServiceName, "kafka1_service_name"},
		{runtimecompose.ValueKafka2ServiceName, "kafka2_service_name"},
		{runtimecompose.ValueKafka3ServiceName, "kafka3_service_name"},
		{runtimecompose.ValueKafka1ContainerName, "kafka1_container_name"},
		{runtimecompose.ValueKafka2ContainerName, "kafka2_container_name"},
		{runtimecompose.ValueKafka3ContainerName, "kafka3_container_name"},
		{runtimecompose.ValueKafka1DataDir, "kafka1_data_dir"},
		{runtimecompose.ValueKafka2DataDir, "kafka2_data_dir"},
		{runtimecompose.ValueKafka3DataDir, "kafka3_data_dir"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("kafka cluster validate %s: must be a non-empty string", field.name)
		}
	}

	portKeys := []struct{ key, name string }{
		{runtimecompose.ValueKafka1Port, "kafka1_port"},
		{runtimecompose.ValueKafka2Port, "kafka2_port"},
		{runtimecompose.ValueKafka3Port, "kafka3_port"},
	}
	seen := make(map[int]string, len(portKeys))
	for _, field := range portKeys {
		port, err := normalizePort(service.Values[field.key])
		if err != nil {
			return fmt.Errorf("kafka cluster validate %s: %w", field.name, err)
		}
		if port <= 0 {
			return fmt.Errorf("kafka cluster validate %s: must be greater than 0", field.name)
		}
		if previous, ok := seen[port]; ok {
			return fmt.Errorf("kafka cluster validate ports: %s and %s must be different", previous, field.name)
		}
		seen[port] = field.name
	}
	return nil
}

func (*clusterSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:             defaultVersion,
		runtimecompose.ValueImage:               "",
		runtimecompose.ValueClusterID:           defaultClusterID,
		runtimecompose.ValueKafka1ServiceName:   "",
		runtimecompose.ValueKafka2ServiceName:   "",
		runtimecompose.ValueKafka3ServiceName:   "",
		runtimecompose.ValueKafka1ContainerName: "",
		runtimecompose.ValueKafka2ContainerName: "",
		runtimecompose.ValueKafka3ContainerName: "",
		runtimecompose.ValueKafka1DataDir:       "",
		runtimecompose.ValueKafka2DataDir:       "",
		runtimecompose.ValueKafka3DataDir:       "",
		runtimecompose.ValueKafka1Port:          defaultKafka1Port,
		runtimecompose.ValueKafka2Port:          defaultKafka2Port,
		runtimecompose.ValueKafka3Port:          defaultKafka3Port,
	}
}
