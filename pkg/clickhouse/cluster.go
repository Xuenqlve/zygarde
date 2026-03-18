package clickhouse

import (
	"fmt"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

const (
	clusterTemplate    = "cluster"
	defaultCH1HTTPPort = 8123
	defaultCH2HTTPPort = 8124
	defaultCH3HTTPPort = 8125
	defaultCH1TCPPort  = 9000
	defaultCH2TCPPort  = 9001
	defaultCH3TCPPort  = 9002
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
		return model.BlueprintService{}, fmt.Errorf("normalize clickhouse cluster version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))

	values[runtimecompose.ValueCH1ServiceName] = defaultStringValue(values[runtimecompose.ValueCH1ServiceName], name+"-ch1")
	values[runtimecompose.ValueCH2ServiceName] = defaultStringValue(values[runtimecompose.ValueCH2ServiceName], name+"-ch2")
	values[runtimecompose.ValueCH3ServiceName] = defaultStringValue(values[runtimecompose.ValueCH3ServiceName], name+"-ch3")
	values[runtimecompose.ValueCH1ContainerName] = defaultStringValue(values[runtimecompose.ValueCH1ContainerName], values[runtimecompose.ValueCH1ServiceName].(string))
	values[runtimecompose.ValueCH2ContainerName] = defaultStringValue(values[runtimecompose.ValueCH2ContainerName], values[runtimecompose.ValueCH2ServiceName].(string))
	values[runtimecompose.ValueCH3ContainerName] = defaultStringValue(values[runtimecompose.ValueCH3ContainerName], values[runtimecompose.ValueCH3ServiceName].(string))
	values[runtimecompose.ValueCH1DataDir] = defaultStringValue(values[runtimecompose.ValueCH1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueCH1ServiceName].(string)))
	values[runtimecompose.ValueCH2DataDir] = defaultStringValue(values[runtimecompose.ValueCH2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueCH2ServiceName].(string)))
	values[runtimecompose.ValueCH3DataDir] = defaultStringValue(values[runtimecompose.ValueCH3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueCH3ServiceName].(string)))

	var err error
	values[runtimecompose.ValueCH1HTTPPort], err = allocateOrReservePort(values[runtimecompose.ValueCH1HTTPPort], hasValue(input.Values, runtimecompose.ValueCH1HTTPPort), defaultCH1HTTPPort, "clickhouse cluster ch1_http_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueCH2HTTPPort], err = allocateOrReservePort(values[runtimecompose.ValueCH2HTTPPort], hasValue(input.Values, runtimecompose.ValueCH2HTTPPort), defaultCH2HTTPPort, "clickhouse cluster ch2_http_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueCH3HTTPPort], err = allocateOrReservePort(values[runtimecompose.ValueCH3HTTPPort], hasValue(input.Values, runtimecompose.ValueCH3HTTPPort), defaultCH3HTTPPort, "clickhouse cluster ch3_http_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueCH1TCPPort], err = allocateOrReservePort(values[runtimecompose.ValueCH1TCPPort], hasValue(input.Values, runtimecompose.ValueCH1TCPPort), defaultCH1TCPPort, "clickhouse cluster ch1_tcp_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueCH2TCPPort], err = allocateOrReservePort(values[runtimecompose.ValueCH2TCPPort], hasValue(input.Values, runtimecompose.ValueCH2TCPPort), defaultCH2TCPPort, "clickhouse cluster ch2_tcp_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueCH3TCPPort], err = allocateOrReservePort(values[runtimecompose.ValueCH3TCPPort], hasValue(input.Values, runtimecompose.ValueCH3TCPPort), defaultCH3TCPPort, "clickhouse cluster ch3_tcp_port")
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

		type nodeSpec struct {
			serviceKey   string
			containerKey string
			dataDirKey   string
			httpPortKey  string
			tcpPortKey   string
			configFile   string
			networkFile  string
		}
		nodes := []nodeSpec{
			{runtimecompose.ValueCH1ServiceName, runtimecompose.ValueCH1ContainerName, runtimecompose.ValueCH1DataDir, runtimecompose.ValueCH1HTTPPort, runtimecompose.ValueCH1TCPPort, "config/ch1/config.d/cluster.xml", "config/ch1/users.d/default-network.xml"},
			{runtimecompose.ValueCH2ServiceName, runtimecompose.ValueCH2ContainerName, runtimecompose.ValueCH2DataDir, runtimecompose.ValueCH2HTTPPort, runtimecompose.ValueCH2TCPPort, "config/ch2/config.d/cluster.xml", "config/ch2/users.d/default-network.xml"},
			{runtimecompose.ValueCH3ServiceName, runtimecompose.ValueCH3ContainerName, runtimecompose.ValueCH3DataDir, runtimecompose.ValueCH3HTTPPort, runtimecompose.ValueCH3TCPPort, "config/ch3/config.d/cluster.xml", "config/ch3/users.d/default-network.xml"},
		}

		nodeNames := []string{
			service.Values[runtimecompose.ValueCH1ServiceName].(string),
			service.Values[runtimecompose.ValueCH2ServiceName].(string),
			service.Values[runtimecompose.ValueCH3ServiceName].(string),
		}
		clusterContexts := make([]runtime.ComposeContext, 0, len(nodes))
		for _, node := range nodes {
			serviceName := service.Values[node.serviceKey].(string)
			containerName := service.Values[node.containerKey].(string)
			dataDir := service.Values[node.dataDirKey].(string)
			httpPort := service.Values[node.httpPortKey].(int)
			tcpPort := service.Values[node.tcpPortKey].(int)
			envKeyPrefix := serviceEnvKeyPrefix(serviceName)

			assets := []runtime.AssetSpec{
				{
					Name:      "clickhouse-cluster-env",
					PathKey:   "env_file",
					Content:   fmt.Sprintf("%s_VERSION=%s\n%s_HTTP_PORT=%d\n%s_TCP_PORT=%d\n", envKeyPrefix, version, envKeyPrefix, httpPort, envKeyPrefix, tcpPort),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "clickhouse-cluster-config",
					FileName:  node.configFile,
					Content:   clickhouseClusterConfig(nodeNames),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeUnique,
				},
			}
			volumes := []runtime.VolumeMount{
				{Source: dataDir, Target: clickhouseTarget},
				{Source: "./" + node.configFile, Target: "/etc/clickhouse-server/config.d/cluster.xml", ReadOnly: true},
			}
			if version == "v25" {
				assets = append(assets, runtime.AssetSpec{
					Name:      "clickhouse-cluster-network",
					FileName:  node.networkFile,
					Content:   clickhouseDefaultNetworkConfig(),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeUnique,
				})
				volumes = append(volumes, runtime.VolumeMount{
					Source:   "./" + node.networkFile,
					Target:   "/etc/clickhouse-server/users.d/default-network.xml",
					ReadOnly: true,
				})
			}

			clusterContexts = append(clusterContexts, runtime.ComposeContext{
				EnvType:     runtimeType,
				ServiceName: serviceName,
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
					Volumes: volumes,
					HealthCheck: &runtime.HealthCheck{
						Test:        []string{"CMD-SHELL", "clickhouse-client -q \"SELECT 1\" >/dev/null 2>&1"},
						Interval:    "5s",
						Timeout:     "5s",
						Retries:     60,
						StartPeriod: "20s",
					},
				},
				Assets:   assets,
				Metadata: tpl.MergeValues(nil, service.Values),
			})
		}

		clusterContexts[0] = addClusterSharedAssets(clusterContexts[0], service, version, image, nodeNames)
		for _, item := range clusterContexts {
			contexts = append(contexts, item)
		}
	}

	return contexts, nil
}

func addClusterSharedAssets(context runtime.ComposeContext, service model.BlueprintService, version, image string, nodeNames []string) runtime.ComposeContext {
	ch1Container := service.Values[runtimecompose.ValueCH1ContainerName].(string)
	ch2Container := service.Values[runtimecompose.ValueCH2ContainerName].(string)
	ch3Container := service.Values[runtimecompose.ValueCH3ContainerName].(string)
	context.Assets = append(context.Assets,
		runtime.AssetSpec{
			Name:      "clickhouse-cluster-build",
			PathKey:   "build_script",
			Content:   fmt.Sprintf("echo \"ClickHouse %s (%s) cluster compose stack started\"\n", service.Name, version),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:    "clickhouse-cluster-check",
			PathKey: "check_script",
			Content: fmt.Sprintf(
				"\"$CONTAINER_ENGINE\" exec %s clickhouse-client -q \"SELECT 1\"\n"+
					"\"$CONTAINER_ENGINE\" exec %s clickhouse-client -q \"SELECT 1\"\n"+
					"\"$CONTAINER_ENGINE\" exec %s clickhouse-client -q \"SELECT 1\"\n"+
					"CNT=\"$(\"$CONTAINER_ENGINE\" exec %s clickhouse-client -q \\\"SELECT count() FROM system.clusters WHERE cluster='zygarde_cluster'\\\" | tr -d '[:space:]')\"\n"+
					"[ \"${CNT:-0}\" -ge 3 ]\n"+
					"OUT=\"$(\"$CONTAINER_ENGINE\" exec %s clickhouse-client -q \\\"SELECT count() FROM remote('%s', system.one)\\\" | tr -d '[:space:]')\"\n"+
					"[ \"$OUT\" = \"%d\" ]\n",
				ch1Container,
				ch2Container,
				ch3Container,
				ch1Container,
				ch1Container,
				strings.Join(nodeNames, ","),
				len(nodeNames),
			),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:      "clickhouse-cluster-readme",
			PathKey:   "readme_file",
			Content:   clickhouseClusterReadme(service, version, image),
			Mode:      0o644,
			MergeMode: runtime.AssetMergeReadme,
		},
	)
	return context
}

func clickhouseClusterConfig(nodeNames []string) string {
	var builder strings.Builder
	builder.WriteString("<clickhouse>\n")
	builder.WriteString("  <remote_servers>\n")
	builder.WriteString("    <zygarde_cluster>\n")
	builder.WriteString("      <shard>\n")
	for _, nodeName := range nodeNames {
		builder.WriteString(fmt.Sprintf("        <replica><host>%s</host><port>9000</port></replica>\n", nodeName))
	}
	builder.WriteString("      </shard>\n")
	builder.WriteString("    </zygarde_cluster>\n")
	builder.WriteString("  </remote_servers>\n")
	builder.WriteString("</clickhouse>\n")
	return builder.String()
}

func clickhouseDefaultNetworkConfig() string {
	return "<clickhouse>\n  <users>\n    <default>\n      <networks>\n        <ip>0.0.0.0/0</ip>\n        <ip>::/0</ip>\n      </networks>\n    </default>\n  </users>\n</clickhouse>\n"
}

func clickhouseClusterReadme(service model.BlueprintService, version, image string) string {
	return fmt.Sprintf(
		"# ClickHouse %s Cluster\n\n- version: %s\n- image: %s\n- ch1 http port: %d\n- ch1 tcp port: %d\n- ch2 http port: %d\n- ch2 tcp port: %d\n- ch3 http port: %d\n- ch3 tcp port: %d\n",
		service.Name,
		version,
		image,
		service.Values[runtimecompose.ValueCH1HTTPPort].(int),
		service.Values[runtimecompose.ValueCH1TCPPort].(int),
		service.Values[runtimecompose.ValueCH2HTTPPort].(int),
		service.Values[runtimecompose.ValueCH2TCPPort].(int),
		service.Values[runtimecompose.ValueCH3HTTPPort].(int),
		service.Values[runtimecompose.ValueCH3TCPPort].(int),
	)
}

func (*clusterSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template == "" {
		return tpl.ErrServiceTemplateRequired
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("clickhouse cluster validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != clusterTemplate {
		return fmt.Errorf("clickhouse cluster validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || strings.TrimSpace(version) == "" {
		return fmt.Errorf("clickhouse cluster validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("clickhouse cluster validate version: %w", err)
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueCH1ServiceName, "ch1_service_name"},
		{runtimecompose.ValueCH2ServiceName, "ch2_service_name"},
		{runtimecompose.ValueCH3ServiceName, "ch3_service_name"},
		{runtimecompose.ValueCH1ContainerName, "ch1_container_name"},
		{runtimecompose.ValueCH2ContainerName, "ch2_container_name"},
		{runtimecompose.ValueCH3ContainerName, "ch3_container_name"},
		{runtimecompose.ValueCH1DataDir, "ch1_data_dir"},
		{runtimecompose.ValueCH2DataDir, "ch2_data_dir"},
		{runtimecompose.ValueCH3DataDir, "ch3_data_dir"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("clickhouse cluster validate %s: must be a non-empty string", field.name)
		}
	}

	seen := map[int]string{}
	for _, field := range []struct {
		key  string
		name string
	}{
		{runtimecompose.ValueCH1HTTPPort, "ch1_http_port"},
		{runtimecompose.ValueCH2HTTPPort, "ch2_http_port"},
		{runtimecompose.ValueCH3HTTPPort, "ch3_http_port"},
		{runtimecompose.ValueCH1TCPPort, "ch1_tcp_port"},
		{runtimecompose.ValueCH2TCPPort, "ch2_tcp_port"},
		{runtimecompose.ValueCH3TCPPort, "ch3_tcp_port"},
	} {
		port, err := normalizePort(service.Values[field.key])
		if err != nil {
			return fmt.Errorf("clickhouse cluster validate %s: %w", field.name, err)
		}
		if port <= 0 {
			return fmt.Errorf("clickhouse cluster validate %s: must be greater than 0", field.name)
		}
		if previous, ok := seen[port]; ok {
			return fmt.Errorf("clickhouse cluster validate ports: %s and %s must be different", previous, field.name)
		}
		seen[port] = field.name
	}

	return nil
}

func (*clusterSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:     defaultVersion,
		runtimecompose.ValueImage:       "",
		runtimecompose.ValueCH1DataDir:  "",
		runtimecompose.ValueCH2DataDir:  "",
		runtimecompose.ValueCH3DataDir:  "",
		runtimecompose.ValueCH1HTTPPort: defaultCH1HTTPPort,
		runtimecompose.ValueCH2HTTPPort: defaultCH2HTTPPort,
		runtimecompose.ValueCH3HTTPPort: defaultCH3HTTPPort,
		runtimecompose.ValueCH1TCPPort:  defaultCH1TCPPort,
		runtimecompose.ValueCH2TCPPort:  defaultCH2TCPPort,
		runtimecompose.ValueCH3TCPPort:  defaultCH3TCPPort,
	}
}
