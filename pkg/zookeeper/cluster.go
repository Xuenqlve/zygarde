package zookeeper

import (
	"fmt"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

const (
	clusterTemplate      = "cluster"
	defaultZK1ClientPort = 2181
	defaultZK2ClientPort = 2182
	defaultZK3ClientPort = 2183
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
		return model.BlueprintService{}, fmt.Errorf("normalize zookeeper cluster version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))

	values[runtimecompose.ValueZK1ServiceName] = defaultStringValue(values[runtimecompose.ValueZK1ServiceName], name+"-zk1")
	values[runtimecompose.ValueZK2ServiceName] = defaultStringValue(values[runtimecompose.ValueZK2ServiceName], name+"-zk2")
	values[runtimecompose.ValueZK3ServiceName] = defaultStringValue(values[runtimecompose.ValueZK3ServiceName], name+"-zk3")
	values[runtimecompose.ValueZK1ContainerName] = defaultStringValue(values[runtimecompose.ValueZK1ContainerName], values[runtimecompose.ValueZK1ServiceName].(string))
	values[runtimecompose.ValueZK2ContainerName] = defaultStringValue(values[runtimecompose.ValueZK2ContainerName], values[runtimecompose.ValueZK2ServiceName].(string))
	values[runtimecompose.ValueZK3ContainerName] = defaultStringValue(values[runtimecompose.ValueZK3ContainerName], values[runtimecompose.ValueZK3ServiceName].(string))
	values[runtimecompose.ValueZK1DataDir] = defaultStringValue(values[runtimecompose.ValueZK1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueZK1ServiceName].(string)))
	values[runtimecompose.ValueZK2DataDir] = defaultStringValue(values[runtimecompose.ValueZK2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueZK2ServiceName].(string)))
	values[runtimecompose.ValueZK3DataDir] = defaultStringValue(values[runtimecompose.ValueZK3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueZK3ServiceName].(string)))
	values[runtimecompose.ValueZK1DatalogDir] = defaultStringValue(values[runtimecompose.ValueZK1DatalogDir], fmt.Sprintf("./datalog/%s", values[runtimecompose.ValueZK1ServiceName].(string)))
	values[runtimecompose.ValueZK2DatalogDir] = defaultStringValue(values[runtimecompose.ValueZK2DatalogDir], fmt.Sprintf("./datalog/%s", values[runtimecompose.ValueZK2ServiceName].(string)))
	values[runtimecompose.ValueZK3DatalogDir] = defaultStringValue(values[runtimecompose.ValueZK3DatalogDir], fmt.Sprintf("./datalog/%s", values[runtimecompose.ValueZK3ServiceName].(string)))

	var err error
	values[runtimecompose.ValueZK1ClientPort], err = allocateOrReservePort(values[runtimecompose.ValueZK1ClientPort], hasValue(input.Values, runtimecompose.ValueZK1ClientPort), defaultZK1ClientPort, "zookeeper cluster zk1_client_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueZK2ClientPort], err = allocateOrReservePort(values[runtimecompose.ValueZK2ClientPort], hasValue(input.Values, runtimecompose.ValueZK2ClientPort), defaultZK2ClientPort, "zookeeper cluster zk2_client_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueZK3ClientPort], err = allocateOrReservePort(values[runtimecompose.ValueZK3ClientPort], hasValue(input.Values, runtimecompose.ValueZK3ClientPort), defaultZK3ClientPort, "zookeeper cluster zk3_client_port")
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
		serverSpec := zookeeperServerSpec(
			service.Values[runtimecompose.ValueZK1ServiceName].(string),
			service.Values[runtimecompose.ValueZK2ServiceName].(string),
			service.Values[runtimecompose.ValueZK3ServiceName].(string),
		)

		type nodeSpec struct {
			id            string
			serviceKey    string
			containerKey  string
			dataDirKey    string
			datalogKey    string
			clientPortKey string
		}
		nodes := []nodeSpec{
			{"1", runtimecompose.ValueZK1ServiceName, runtimecompose.ValueZK1ContainerName, runtimecompose.ValueZK1DataDir, runtimecompose.ValueZK1DatalogDir, runtimecompose.ValueZK1ClientPort},
			{"2", runtimecompose.ValueZK2ServiceName, runtimecompose.ValueZK2ContainerName, runtimecompose.ValueZK2DataDir, runtimecompose.ValueZK2DatalogDir, runtimecompose.ValueZK2ClientPort},
			{"3", runtimecompose.ValueZK3ServiceName, runtimecompose.ValueZK3ContainerName, runtimecompose.ValueZK3DataDir, runtimecompose.ValueZK3DatalogDir, runtimecompose.ValueZK3ClientPort},
		}

		clusterContexts := make([]runtime.ComposeContext, 0, len(nodes))
		for _, node := range nodes {
			serviceName := service.Values[node.serviceKey].(string)
			containerName := service.Values[node.containerKey].(string)
			dataDir := service.Values[node.dataDirKey].(string)
			datalogDir := service.Values[node.datalogKey].(string)
			clientPort := service.Values[node.clientPortKey].(int)
			envKeyPrefix := serviceEnvKeyPrefix(serviceName)

			clusterContexts = append(clusterContexts, runtime.ComposeContext{
				EnvType:     runtimeType,
				ServiceName: serviceName,
				Middleware:  service.Middleware,
				Template:    service.Template,
				Service: runtime.ServiceSpec{
					Image:         image,
					ContainerName: containerName,
					Restart:       "unless-stopped",
					Environment: map[string]string{
						"ZOO_MY_ID":                  node.id,
						"ZOO_SERVERS":                serverSpec,
						"ZOO_4LW_COMMANDS_WHITELIST": "ruok,mntr,srvr,stat,conf,isro",
					},
					Ports: []runtime.PortBinding{
						{HostPort: clientPort, ContainerPort: 2181, Protocol: "tcp"},
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
						Name:      "zookeeper-cluster-env",
						PathKey:   "env_file",
						Content:   fmt.Sprintf("%s_VERSION=%s\n%s_CLIENT_PORT=%d\n", envKeyPrefix, version, envKeyPrefix, clientPort),
						Mode:      0o644,
						MergeMode: runtime.AssetMergeEnv,
					},
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			})
		}

		clusterContexts[0] = addClusterSharedAssets(clusterContexts[0], service, version, image)
		for _, item := range clusterContexts {
			contexts = append(contexts, item)
		}
	}
	return contexts, nil
}

func addClusterSharedAssets(context runtime.ComposeContext, service model.BlueprintService, version, image string) runtime.ComposeContext {
	zk1Container := service.Values[runtimecompose.ValueZK1ContainerName].(string)
	zk2Container := service.Values[runtimecompose.ValueZK2ContainerName].(string)
	zk3Container := service.Values[runtimecompose.ValueZK3ContainerName].(string)
	zk1Service := service.Values[runtimecompose.ValueZK1ServiceName].(string)
	zk2Service := service.Values[runtimecompose.ValueZK2ServiceName].(string)
	zk3Service := service.Values[runtimecompose.ValueZK3ServiceName].(string)

	context.Assets = append(context.Assets,
		runtime.AssetSpec{
			Name:      "zookeeper-cluster-build",
			PathKey:   "build_script",
			Content:   fmt.Sprintf("echo \"ZooKeeper %s (%s) cluster compose stack started\"\n", service.Name, version),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:    "zookeeper-cluster-check",
			PathKey: "check_script",
			Content: fmt.Sprintf(
				"for c in %s %s %s; do\n"+
					"  R=\"$({ echo ruok | \"$CONTAINER_ENGINE\" exec -i \"$c\" /bin/bash -lc 'cat | nc 127.0.0.1 2181'; } | tr -d '\\r')\"\n"+
					"  [ \"$R\" = \"imok\" ]\n"+
					"done\n"+
					"ROLE_COUNT=\"$(\"$CONTAINER_ENGINE\" exec %s /bin/bash -lc \\\"for h in %s %s %s; do echo stat | nc \\\\\\$h 2181 | grep Mode; done\\\" | wc -l | tr -d '[:space:]')\"\n"+
					"[ \"$ROLE_COUNT\" = \"3\" ]\n"+
					"\"$CONTAINER_ENGINE\" exec %s /bin/bash -lc \"zkCli.sh -server %s:2181 create /zygarde_cluster_smoke ok >/tmp/zk_cluster.out 2>&1 || true\"\n"+
					"OUT=\"$(\"$CONTAINER_ENGINE\" exec %s /bin/bash -lc \"zkCli.sh -server %s:2181 get /zygarde_cluster_smoke 2>/dev/null | grep -E '^ok$' | head -n1\" | tr -d '\\r')\"\n"+
					"[ \"$OUT\" = \"ok\" ]\n",
				zk1Container, zk2Container, zk3Container,
				zk1Container, zk1Service, zk2Service, zk3Service,
				zk1Container, zk1Service,
				zk3Container, zk3Service,
			),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:      "zookeeper-cluster-readme",
			PathKey:   "readme_file",
			Content:   zookeeperClusterReadme(service, version, image),
			Mode:      0o644,
			MergeMode: runtime.AssetMergeReadme,
		},
	)
	return context
}

func (*clusterSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template == "" {
		return tpl.ErrServiceTemplateRequired
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("zookeeper cluster validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != clusterTemplate {
		return fmt.Errorf("zookeeper cluster validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || strings.TrimSpace(version) == "" {
		return fmt.Errorf("zookeeper cluster validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("zookeeper cluster validate version: %w", err)
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueZK1ServiceName, "zk1_service_name"},
		{runtimecompose.ValueZK2ServiceName, "zk2_service_name"},
		{runtimecompose.ValueZK3ServiceName, "zk3_service_name"},
		{runtimecompose.ValueZK1ContainerName, "zk1_container_name"},
		{runtimecompose.ValueZK2ContainerName, "zk2_container_name"},
		{runtimecompose.ValueZK3ContainerName, "zk3_container_name"},
		{runtimecompose.ValueZK1DataDir, "zk1_data_dir"},
		{runtimecompose.ValueZK2DataDir, "zk2_data_dir"},
		{runtimecompose.ValueZK3DataDir, "zk3_data_dir"},
		{runtimecompose.ValueZK1DatalogDir, "zk1_datalog_dir"},
		{runtimecompose.ValueZK2DatalogDir, "zk2_datalog_dir"},
		{runtimecompose.ValueZK3DatalogDir, "zk3_datalog_dir"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("zookeeper cluster validate %s: must be a non-empty string", field.name)
		}
	}

	seen := map[int]string{}
	for _, field := range []struct {
		key  string
		name string
	}{
		{runtimecompose.ValueZK1ClientPort, "zk1_client_port"},
		{runtimecompose.ValueZK2ClientPort, "zk2_client_port"},
		{runtimecompose.ValueZK3ClientPort, "zk3_client_port"},
	} {
		port, err := normalizePort(service.Values[field.key])
		if err != nil {
			return fmt.Errorf("zookeeper cluster validate %s: %w", field.name, err)
		}
		if port <= 0 {
			return fmt.Errorf("zookeeper cluster validate %s: must be greater than 0", field.name)
		}
		if previous, ok := seen[port]; ok {
			return fmt.Errorf("zookeeper cluster validate ports: %s and %s must be different", previous, field.name)
		}
		seen[port] = field.name
	}

	return nil
}

func (*clusterSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:       defaultVersion,
		runtimecompose.ValueImage:         "",
		runtimecompose.ValueZK1DataDir:    "",
		runtimecompose.ValueZK2DataDir:    "",
		runtimecompose.ValueZK3DataDir:    "",
		runtimecompose.ValueZK1DatalogDir: "",
		runtimecompose.ValueZK2DatalogDir: "",
		runtimecompose.ValueZK3DatalogDir: "",
		runtimecompose.ValueZK1ClientPort: defaultZK1ClientPort,
		runtimecompose.ValueZK2ClientPort: defaultZK2ClientPort,
		runtimecompose.ValueZK3ClientPort: defaultZK3ClientPort,
	}
}

func zookeeperServerSpec(zk1, zk2, zk3 string) string {
	return fmt.Sprintf(
		"server.1=%s:2888:3888;2181 server.2=%s:2888:3888;2181 server.3=%s:2888:3888;2181",
		zk1, zk2, zk3,
	)
}

func zookeeperClusterReadme(service model.BlueprintService, version, image string) string {
	return fmt.Sprintf(
		"# ZooKeeper %s Cluster\n\n- version: %s\n- image: %s\n- zk1 client port: %d\n- zk2 client port: %d\n- zk3 client port: %d\n",
		service.Name,
		version,
		image,
		service.Values[runtimecompose.ValueZK1ClientPort].(int),
		service.Values[runtimecompose.ValueZK2ClientPort].(int),
		service.Values[runtimecompose.ValueZK3ClientPort].(int),
	)
}
