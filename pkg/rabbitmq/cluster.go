package rabbitmq

import (
	"fmt"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

const (
	clusterTemplate          = "cluster"
	defaultRabbit1AMQPPort   = 5672
	defaultRabbit2AMQPPort   = 5673
	defaultRabbit3AMQPPort   = 5674
	defaultRabbit1ManagePort = 15672
	defaultRabbit2ManagePort = 15673
	defaultRabbit3ManagePort = 15674
)

// NewClusterSpec returns the default RabbitMQ cluster middleware spec.
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
		return model.BlueprintService{}, fmt.Errorf("normalize rabbitmq cluster version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version

	image := defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueImage] = image
	values[runtimecompose.ValueDefaultUser] = defaultStringValue(values[runtimecompose.ValueDefaultUser], "admin")
	values[runtimecompose.ValueDefaultPass] = defaultStringValue(values[runtimecompose.ValueDefaultPass], "admin123")
	values[runtimecompose.ValueErlangCookie] = defaultStringValue(values[runtimecompose.ValueErlangCookie], "rabbitmq-cookie")

	values[runtimecompose.ValueRabbit1ServiceName] = defaultStringValue(values[runtimecompose.ValueRabbit1ServiceName], name+"-rabbit1")
	values[runtimecompose.ValueRabbit2ServiceName] = defaultStringValue(values[runtimecompose.ValueRabbit2ServiceName], name+"-rabbit2")
	values[runtimecompose.ValueRabbit3ServiceName] = defaultStringValue(values[runtimecompose.ValueRabbit3ServiceName], name+"-rabbit3")
	values[runtimecompose.ValueRabbit1ContainerName] = defaultStringValue(values[runtimecompose.ValueRabbit1ContainerName], values[runtimecompose.ValueRabbit1ServiceName].(string))
	values[runtimecompose.ValueRabbit2ContainerName] = defaultStringValue(values[runtimecompose.ValueRabbit2ContainerName], values[runtimecompose.ValueRabbit2ServiceName].(string))
	values[runtimecompose.ValueRabbit3ContainerName] = defaultStringValue(values[runtimecompose.ValueRabbit3ContainerName], values[runtimecompose.ValueRabbit3ServiceName].(string))
	values[runtimecompose.ValueRabbit1DataDir] = defaultStringValue(values[runtimecompose.ValueRabbit1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueRabbit1ServiceName].(string)))
	values[runtimecompose.ValueRabbit2DataDir] = defaultStringValue(values[runtimecompose.ValueRabbit2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueRabbit2ServiceName].(string)))
	values[runtimecompose.ValueRabbit3DataDir] = defaultStringValue(values[runtimecompose.ValueRabbit3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueRabbit3ServiceName].(string)))

	var err error
	values[runtimecompose.ValueRabbit1AMQPPort], err = allocateOrReservePort(
		values[runtimecompose.ValueRabbit1AMQPPort],
		hasValue(input.Values, runtimecompose.ValueRabbit1AMQPPort),
		defaultRabbit1AMQPPort,
		"rabbitmq cluster rabbit1_amqp_port",
	)
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueRabbit2AMQPPort], err = allocateOrReservePort(
		values[runtimecompose.ValueRabbit2AMQPPort],
		hasValue(input.Values, runtimecompose.ValueRabbit2AMQPPort),
		defaultRabbit2AMQPPort,
		"rabbitmq cluster rabbit2_amqp_port",
	)
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueRabbit3AMQPPort], err = allocateOrReservePort(
		values[runtimecompose.ValueRabbit3AMQPPort],
		hasValue(input.Values, runtimecompose.ValueRabbit3AMQPPort),
		defaultRabbit3AMQPPort,
		"rabbitmq cluster rabbit3_amqp_port",
	)
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueRabbit1ManagementPort], err = allocateOrReservePort(
		values[runtimecompose.ValueRabbit1ManagementPort],
		hasValue(input.Values, runtimecompose.ValueRabbit1ManagementPort),
		defaultRabbit1ManagePort,
		"rabbitmq cluster rabbit1_management_port",
	)
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueRabbit2ManagementPort], err = allocateOrReservePort(
		values[runtimecompose.ValueRabbit2ManagementPort],
		hasValue(input.Values, runtimecompose.ValueRabbit2ManagementPort),
		defaultRabbit2ManagePort,
		"rabbitmq cluster rabbit2_management_port",
	)
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueRabbit3ManagementPort], err = allocateOrReservePort(
		values[runtimecompose.ValueRabbit3ManagementPort],
		hasValue(input.Values, runtimecompose.ValueRabbit3ManagementPort),
		defaultRabbit3ManagePort,
		"rabbitmq cluster rabbit3_management_port",
	)
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

		version := service.Values[runtimecompose.ValueVersion].(string)
		image := service.Values[runtimecompose.ValueImage].(string)
		defaultUser := service.Values[runtimecompose.ValueDefaultUser].(string)
		defaultPass := service.Values[runtimecompose.ValueDefaultPass].(string)
		cookie := service.Values[runtimecompose.ValueErlangCookie].(string)

		type nodeSpec struct {
			serviceKey    string
			containerKey  string
			dataDirKey    string
			amqpPortKey   string
			managePortKey string
			hostname      string
			nodename      string
		}
		nodes := []nodeSpec{
			{runtimecompose.ValueRabbit1ServiceName, runtimecompose.ValueRabbit1ContainerName, runtimecompose.ValueRabbit1DataDir, runtimecompose.ValueRabbit1AMQPPort, runtimecompose.ValueRabbit1ManagementPort, "rabbit1", "rabbit@rabbit1"},
			{runtimecompose.ValueRabbit2ServiceName, runtimecompose.ValueRabbit2ContainerName, runtimecompose.ValueRabbit2DataDir, runtimecompose.ValueRabbit2AMQPPort, runtimecompose.ValueRabbit2ManagementPort, "rabbit2", "rabbit@rabbit2"},
			{runtimecompose.ValueRabbit3ServiceName, runtimecompose.ValueRabbit3ContainerName, runtimecompose.ValueRabbit3DataDir, runtimecompose.ValueRabbit3AMQPPort, runtimecompose.ValueRabbit3ManagementPort, "rabbit3", "rabbit@rabbit3"},
		}
		confFile := fmt.Sprintf("conf/%s-rabbitmq.conf", service.Name)

		confContent := "cluster_formation.peer_discovery_backend = classic_config\n" +
			"cluster_formation.classic_config.nodes.1 = rabbit@rabbit1\n" +
			"cluster_formation.classic_config.nodes.2 = rabbit@rabbit2\n" +
			"cluster_formation.classic_config.nodes.3 = rabbit@rabbit3\n" +
			"cluster_partition_handling = autoheal\n" +
			"queue_master_locator = min-masters\n"

		clusterContexts := make([]runtime.ComposeContext, 0, len(nodes))
		for _, node := range nodes {
			serviceName := service.Values[node.serviceKey].(string)
			containerName := service.Values[node.containerKey].(string)
			dataDir := service.Values[node.dataDirKey].(string)
			amqpPort := service.Values[node.amqpPortKey].(int)
			managePort := service.Values[node.managePortKey].(int)
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
						"RABBITMQ_DEFAULT_USER":  defaultUser,
						"RABBITMQ_DEFAULT_PASS":  defaultPass,
						"RABBITMQ_ERLANG_COOKIE": cookie,
						"RABBITMQ_NODENAME":      node.nodename,
					},
					Ports: []runtime.PortBinding{
						{HostPort: amqpPort, ContainerPort: 5672, Protocol: "tcp"},
						{HostPort: managePort, ContainerPort: 15672, Protocol: "tcp"},
					},
					Volumes: []runtime.VolumeMount{
						{Source: dataDir, Target: "/var/lib/rabbitmq"},
						{Source: "./" + confFile, Target: "/etc/rabbitmq/rabbitmq.conf", ReadOnly: true},
					},
					HealthCheck: &runtime.HealthCheck{
						Test:        []string{"CMD-SHELL", "rabbitmq-diagnostics -q ping"},
						Interval:    "5s",
						Timeout:     "5s",
						Retries:     60,
						StartPeriod: "20s",
					},
				},
				Assets: []runtime.AssetSpec{
					{
						Name:    "rabbitmq-cluster-env",
						PathKey: "env_file",
						Content: fmt.Sprintf(
							"%s_VERSION=%s\n%s_AMQP_PORT=%d\n%s_MANAGEMENT_PORT=%d\n%s_DEFAULT_USER=%s\n%s_DEFAULT_PASS=%s\n%s_ERLANG_COOKIE=%s\n",
							envKeyPrefix, version,
							envKeyPrefix, amqpPort,
							envKeyPrefix, managePort,
							envKeyPrefix, defaultUser,
							envKeyPrefix, defaultPass,
							envKeyPrefix, cookie,
						),
						Mode:      0o644,
						MergeMode: runtime.AssetMergeEnv,
					},
					{
						Name:      "rabbitmq-cluster-config",
						FileName:  confFile,
						Content:   confContent,
						Mode:      0o644,
						MergeMode: runtime.AssetMergeUnique,
					},
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			})
		}

		container1 := service.Values[runtimecompose.ValueRabbit1ContainerName].(string)
		container2 := service.Values[runtimecompose.ValueRabbit2ContainerName].(string)
		container3 := service.Values[runtimecompose.ValueRabbit3ContainerName].(string)
		clusterContexts[0] = addClusterSharedAssets(clusterContexts[0], service, version, image, container1, container2, container3)
		for _, contextItem := range clusterContexts {
			contexts = append(contexts, contextItem)
		}
	}
	return contexts, nil
}

func addClusterSharedAssets(context runtime.ComposeContext, service model.BlueprintService, version, image, container1, container2, container3 string) runtime.ComposeContext {
	context.Assets = append(context.Assets,
		runtime.AssetSpec{
			Name:      "rabbitmq-cluster-build",
			PathKey:   "build_script",
			Content:   fmt.Sprintf("echo \"RabbitMQ %s (%s) cluster compose stack started\"\n", service.Name, version),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:    "rabbitmq-cluster-check",
			PathKey: "check_script",
			Content: fmt.Sprintf(
				"STATUS=\"$(\"$CONTAINER_ENGINE\" exec %s rabbitmqctl cluster_status --formatter json)\"\n"+
					"echo \"$STATUS\" | grep -q 'rabbit@rabbit1'\n"+
					"echo \"$STATUS\" | grep -q 'rabbit@rabbit2'\n"+
					"echo \"$STATUS\" | grep -q 'rabbit@rabbit3'\n"+
					"\"$CONTAINER_ENGINE\" exec %s rabbitmq-diagnostics -q cluster_status\n"+
					"\"$CONTAINER_ENGINE\" exec %s rabbitmq-diagnostics -q ping\n"+
					"\"$CONTAINER_ENGINE\" exec %s rabbitmq-diagnostics -q ping\n",
				container1, container1, container2, container3,
			),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:      "rabbitmq-cluster-readme",
			PathKey:   "readme_file",
			Content:   rabbitmqClusterReadme(service, version, image),
			Mode:      0o644,
			MergeMode: runtime.AssetMergeReadme,
		},
	)
	return context
}

func rabbitmqClusterReadme(service model.BlueprintService, version, image string) string {
	return fmt.Sprintf(
		"# RabbitMQ %s Cluster\n\n- version: %s\n- image: %s\n- rabbit1 amqp port: %d\n- rabbit1 management port: %d\n- rabbit2 amqp port: %d\n- rabbit2 management port: %d\n- rabbit3 amqp port: %d\n- rabbit3 management port: %d\n",
		service.Name,
		version,
		image,
		service.Values[runtimecompose.ValueRabbit1AMQPPort].(int),
		service.Values[runtimecompose.ValueRabbit1ManagementPort].(int),
		service.Values[runtimecompose.ValueRabbit2AMQPPort].(int),
		service.Values[runtimecompose.ValueRabbit2ManagementPort].(int),
		service.Values[runtimecompose.ValueRabbit3AMQPPort].(int),
		service.Values[runtimecompose.ValueRabbit3ManagementPort].(int),
	)
}

func (*clusterSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template != clusterTemplate {
		return fmt.Errorf("rabbitmq cluster validate: unexpected template %q", service.Template)
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("rabbitmq cluster validate: unexpected middleware %q", service.Middleware)
	}
	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("rabbitmq cluster validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("rabbitmq cluster validate version: %w", err)
	}

	stringFields := []struct{ key, name string }{
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueDefaultUser, "default_user"},
		{runtimecompose.ValueDefaultPass, "default_pass"},
		{runtimecompose.ValueErlangCookie, "erlang_cookie"},
		{runtimecompose.ValueRabbit1ServiceName, "rabbit1_service_name"},
		{runtimecompose.ValueRabbit2ServiceName, "rabbit2_service_name"},
		{runtimecompose.ValueRabbit3ServiceName, "rabbit3_service_name"},
		{runtimecompose.ValueRabbit1ContainerName, "rabbit1_container_name"},
		{runtimecompose.ValueRabbit2ContainerName, "rabbit2_container_name"},
		{runtimecompose.ValueRabbit3ContainerName, "rabbit3_container_name"},
		{runtimecompose.ValueRabbit1DataDir, "rabbit1_data_dir"},
		{runtimecompose.ValueRabbit2DataDir, "rabbit2_data_dir"},
		{runtimecompose.ValueRabbit3DataDir, "rabbit3_data_dir"},
	}
	for _, field := range stringFields {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("rabbitmq cluster validate %s: must be a non-empty string", field.name)
		}
	}

	portKeys := []struct{ key, name string }{
		{runtimecompose.ValueRabbit1AMQPPort, "rabbit1_amqp_port"},
		{runtimecompose.ValueRabbit2AMQPPort, "rabbit2_amqp_port"},
		{runtimecompose.ValueRabbit3AMQPPort, "rabbit3_amqp_port"},
		{runtimecompose.ValueRabbit1ManagementPort, "rabbit1_management_port"},
		{runtimecompose.ValueRabbit2ManagementPort, "rabbit2_management_port"},
		{runtimecompose.ValueRabbit3ManagementPort, "rabbit3_management_port"},
	}
	seen := make(map[int]string, len(portKeys))
	for _, field := range portKeys {
		port, err := normalizePort(service.Values[field.key])
		if err != nil {
			return fmt.Errorf("rabbitmq cluster validate %s: %w", field.name, err)
		}
		if port <= 0 {
			return fmt.Errorf("rabbitmq cluster validate %s: must be greater than 0", field.name)
		}
		if previous, ok := seen[port]; ok {
			return fmt.Errorf("rabbitmq cluster validate ports: %s and %s must be different", previous, field.name)
		}
		seen[port] = field.name
	}
	return nil
}

func (*clusterSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:               defaultVersion,
		runtimecompose.ValueImage:                 "",
		runtimecompose.ValueDefaultUser:           "admin",
		runtimecompose.ValueDefaultPass:           "admin123",
		runtimecompose.ValueErlangCookie:          "rabbitmq-cookie",
		runtimecompose.ValueRabbit1ServiceName:    "",
		runtimecompose.ValueRabbit2ServiceName:    "",
		runtimecompose.ValueRabbit3ServiceName:    "",
		runtimecompose.ValueRabbit1ContainerName:  "",
		runtimecompose.ValueRabbit2ContainerName:  "",
		runtimecompose.ValueRabbit3ContainerName:  "",
		runtimecompose.ValueRabbit1DataDir:        "",
		runtimecompose.ValueRabbit2DataDir:        "",
		runtimecompose.ValueRabbit3DataDir:        "",
		runtimecompose.ValueRabbit1AMQPPort:       defaultRabbit1AMQPPort,
		runtimecompose.ValueRabbit2AMQPPort:       defaultRabbit2AMQPPort,
		runtimecompose.ValueRabbit3AMQPPort:       defaultRabbit3AMQPPort,
		runtimecompose.ValueRabbit1ManagementPort: defaultRabbit1ManagePort,
		runtimecompose.ValueRabbit2ManagementPort: defaultRabbit2ManagePort,
		runtimecompose.ValueRabbit3ManagementPort: defaultRabbit3ManagePort,
	}
}
