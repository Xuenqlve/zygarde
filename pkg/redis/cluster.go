package redis

import (
	"fmt"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

const (
	clusterTemplate     = "cluster"
	defaultNode1Port    = 7001
	defaultNode2Port    = 7002
	defaultNode3Port    = 7003
	defaultNode1BusPort = 17001
	defaultNode2BusPort = 17002
	defaultNode3BusPort = 17003
)

// NewClusterSpec returns the default Redis cluster middleware spec.
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
		return model.BlueprintService{}, fmt.Errorf("normalize redis cluster version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))

	values[runtimecompose.ValueNode1ServiceName] = defaultStringValue(values[runtimecompose.ValueNode1ServiceName], fmt.Sprintf("%s-node-1", name))
	values[runtimecompose.ValueNode2ServiceName] = defaultStringValue(values[runtimecompose.ValueNode2ServiceName], fmt.Sprintf("%s-node-2", name))
	values[runtimecompose.ValueNode3ServiceName] = defaultStringValue(values[runtimecompose.ValueNode3ServiceName], fmt.Sprintf("%s-node-3", name))
	values[runtimecompose.ValueNode1ContainerName] = defaultStringValue(values[runtimecompose.ValueNode1ContainerName], values[runtimecompose.ValueNode1ServiceName].(string))
	values[runtimecompose.ValueNode2ContainerName] = defaultStringValue(values[runtimecompose.ValueNode2ContainerName], values[runtimecompose.ValueNode2ServiceName].(string))
	values[runtimecompose.ValueNode3ContainerName] = defaultStringValue(values[runtimecompose.ValueNode3ContainerName], values[runtimecompose.ValueNode3ServiceName].(string))
	values[runtimecompose.ValueNode1DataDir] = defaultStringValue(values[runtimecompose.ValueNode1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueNode1ServiceName].(string)))
	values[runtimecompose.ValueNode2DataDir] = defaultStringValue(values[runtimecompose.ValueNode2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueNode2ServiceName].(string)))
	values[runtimecompose.ValueNode3DataDir] = defaultStringValue(values[runtimecompose.ValueNode3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueNode3ServiceName].(string)))

	specs := []struct {
		key      string
		fallback int
		name     string
	}{
		{runtimecompose.ValueNode1Port, defaultNode1Port, "redis cluster node_1_port"},
		{runtimecompose.ValueNode2Port, defaultNode2Port, "redis cluster node_2_port"},
		{runtimecompose.ValueNode3Port, defaultNode3Port, "redis cluster node_3_port"},
		{runtimecompose.ValueNode1BusPort, defaultNode1BusPort, "redis cluster node_1_bus_port"},
		{runtimecompose.ValueNode2BusPort, defaultNode2BusPort, "redis cluster node_2_bus_port"},
		{runtimecompose.ValueNode3BusPort, defaultNode3BusPort, "redis cluster node_3_bus_port"},
	}
	for _, spec := range specs {
		port, err := allocateOrReservePort(values[spec.key], hasValue(input.Values, spec.key), spec.fallback, spec.name)
		if err != nil {
			return model.BlueprintService{}, err
		}
		values[spec.key] = port
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
		envKeyPrefix := serviceEnvKeyPrefix(service.Name)
		nodes := []struct {
			serviceNameKey   string
			containerNameKey string
			dataDirKey       string
			portKey          string
			busPortKey       string
		}{
			{runtimecompose.ValueNode1ServiceName, runtimecompose.ValueNode1ContainerName, runtimecompose.ValueNode1DataDir, runtimecompose.ValueNode1Port, runtimecompose.ValueNode1BusPort},
			{runtimecompose.ValueNode2ServiceName, runtimecompose.ValueNode2ContainerName, runtimecompose.ValueNode2DataDir, runtimecompose.ValueNode2Port, runtimecompose.ValueNode2BusPort},
			{runtimecompose.ValueNode3ServiceName, runtimecompose.ValueNode3ContainerName, runtimecompose.ValueNode3DataDir, runtimecompose.ValueNode3Port, runtimecompose.ValueNode3BusPort},
		}

		for index, node := range nodes {
			serviceName := service.Values[node.serviceNameKey].(string)
			containerName := service.Values[node.containerNameKey].(string)
			dataDir := service.Values[node.dataDirKey].(string)
			port, err := normalizePort(service.Values[node.portKey])
			if err != nil {
				return nil, fmt.Errorf("redis cluster build runtime context port: %w", err)
			}
			busPort, err := normalizePort(service.Values[node.busPortKey])
			if err != nil {
				return nil, fmt.Errorf("redis cluster build runtime context bus port: %w", err)
			}

			context := runtime.ComposeContext{
				EnvType:     runtimeType,
				ServiceName: serviceName,
				Middleware:  service.Middleware,
				Template:    service.Template,
				Service: runtime.ServiceSpec{
					Image:         image,
					ContainerName: containerName,
					Restart:       "unless-stopped",
					Ports: []runtime.PortBinding{
						{HostPort: port, ContainerPort: port, Protocol: "tcp"},
						{HostPort: busPort, ContainerPort: busPort, Protocol: "tcp"},
					},
					Volumes: []runtime.VolumeMount{{Source: dataDir, Target: "/data"}},
					Command: []string{
						"redis-server",
						"--port", fmt.Sprintf("%d", port),
						"--cluster-enabled", "yes",
						"--cluster-config-file", "nodes.conf",
						"--cluster-node-timeout", "5000",
						"--appendonly", "yes",
					},
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			}
			if index == 0 {
				context.Assets = []runtime.AssetSpec{
					{
						Name:    "redis-cluster-env",
						PathKey: "env_file",
						Content: fmt.Sprintf(
							"%s_VERSION=%s\n%s_IMAGE=%s\n%s_NODE_1_PORT=%d\n%s_NODE_1_BUS_PORT=%d\n%s_NODE_2_PORT=%d\n%s_NODE_2_BUS_PORT=%d\n%s_NODE_3_PORT=%d\n%s_NODE_3_BUS_PORT=%d\n",
							envKeyPrefix, version,
							envKeyPrefix, image,
							envKeyPrefix, service.Values[runtimecompose.ValueNode1Port].(int),
							envKeyPrefix, service.Values[runtimecompose.ValueNode1BusPort].(int),
							envKeyPrefix, service.Values[runtimecompose.ValueNode2Port].(int),
							envKeyPrefix, service.Values[runtimecompose.ValueNode2BusPort].(int),
							envKeyPrefix, service.Values[runtimecompose.ValueNode3Port].(int),
							envKeyPrefix, service.Values[runtimecompose.ValueNode3BusPort].(int),
						),
						Mode:      0o644,
						MergeMode: runtime.AssetMergeEnv,
					},
					{
						Name:      "redis-cluster-build",
						PathKey:   "build_script",
						Content:   redisClusterBuildScript(service.Name, service.Values[runtimecompose.ValueNode1ContainerName].(string), service.Values[runtimecompose.ValueNode2ContainerName].(string), service.Values[runtimecompose.ValueNode3ContainerName].(string), service.Values[runtimecompose.ValueNode1Port].(int), service.Values[runtimecompose.ValueNode2Port].(int), service.Values[runtimecompose.ValueNode3Port].(int)),
						Mode:      0o755,
						MergeMode: runtime.AssetMergeScript,
					},
					{
						Name:      "redis-cluster-check",
						PathKey:   "check_script",
						Content:   redisClusterCheckScript(service.Name, service.Values[runtimecompose.ValueNode1ContainerName].(string), service.Values[runtimecompose.ValueNode2ContainerName].(string), service.Values[runtimecompose.ValueNode3ContainerName].(string), service.Values[runtimecompose.ValueNode1Port].(int), service.Values[runtimecompose.ValueNode2Port].(int), service.Values[runtimecompose.ValueNode3Port].(int)),
						Mode:      0o755,
						MergeMode: runtime.AssetMergeScript,
					},
					{
						Name:      "redis-cluster-readme",
						PathKey:   "readme_file",
						Content:   redisClusterReadme(service.Name, version, image, service.Values[runtimecompose.ValueNode1Port].(int), service.Values[runtimecompose.ValueNode2Port].(int), service.Values[runtimecompose.ValueNode3Port].(int)),
						Mode:      0o644,
						MergeMode: runtime.AssetMergeReadme,
					},
				}
			}

			contexts = append(contexts, context)
		}
	}

	return contexts, nil
}

func (*clusterSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template == "" {
		return tpl.ErrServiceTemplateRequired
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("redis cluster validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != clusterTemplate {
		return fmt.Errorf("redis cluster validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("redis cluster validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("redis cluster validate version: %w", err)
	}

	stringFields := []struct{ key, name string }{
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueNode1ServiceName, "node_1_service_name"},
		{runtimecompose.ValueNode2ServiceName, "node_2_service_name"},
		{runtimecompose.ValueNode3ServiceName, "node_3_service_name"},
		{runtimecompose.ValueNode1ContainerName, "node_1_container_name"},
		{runtimecompose.ValueNode2ContainerName, "node_2_container_name"},
		{runtimecompose.ValueNode3ContainerName, "node_3_container_name"},
		{runtimecompose.ValueNode1DataDir, "node_1_data_dir"},
		{runtimecompose.ValueNode2DataDir, "node_2_data_dir"},
		{runtimecompose.ValueNode3DataDir, "node_3_data_dir"},
	}
	for _, field := range stringFields {
		value, ok := service.Values[field.key].(string)
		if !ok || value == "" {
			return fmt.Errorf("redis cluster validate %s: must be a non-empty string", field.name)
		}
	}

	portKeys := []struct{ key, name string }{
		{runtimecompose.ValueNode1Port, "node_1_port"},
		{runtimecompose.ValueNode2Port, "node_2_port"},
		{runtimecompose.ValueNode3Port, "node_3_port"},
		{runtimecompose.ValueNode1BusPort, "node_1_bus_port"},
		{runtimecompose.ValueNode2BusPort, "node_2_bus_port"},
		{runtimecompose.ValueNode3BusPort, "node_3_bus_port"},
	}
	seen := map[int]string{}
	for _, field := range portKeys {
		port, err := normalizePort(service.Values[field.key])
		if err != nil {
			return fmt.Errorf("redis cluster validate %s: %w", field.name, err)
		}
		if port <= 0 {
			return fmt.Errorf("redis cluster validate %s: must be greater than 0", field.name)
		}
		if prev, ok := seen[port]; ok {
			return fmt.Errorf("redis cluster validate ports: %s conflicts with %s", field.name, prev)
		}
		seen[port] = field.name
	}

	return nil
}

func (*clusterSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:      defaultVersion,
		runtimecompose.ValueImage:        "",
		runtimecompose.ValueNode1DataDir: "",
		runtimecompose.ValueNode2DataDir: "",
		runtimecompose.ValueNode3DataDir: "",
		runtimecompose.ValueNode1Port:    defaultNode1Port,
		runtimecompose.ValueNode2Port:    defaultNode2Port,
		runtimecompose.ValueNode3Port:    defaultNode3Port,
		runtimecompose.ValueNode1BusPort: defaultNode1BusPort,
		runtimecompose.ValueNode2BusPort: defaultNode2BusPort,
		runtimecompose.ValueNode3BusPort: defaultNode3BusPort,
	}
}

func redisClusterBuildScript(name, node1, node2, node3 string, port1, port2, port3 int) string {
	return fmt.Sprintf(`echo "[redis cluster] waiting for %s nodes"
sleep 8

IP1="$("$CONTAINER_ENGINE" inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' %s)"
IP2="$("$CONTAINER_ENGINE" inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' %s)"
IP3="$("$CONTAINER_ENGINE" inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' %s)"

echo "[redis cluster] creating cluster for %s"
"$CONTAINER_ENGINE" exec -i %s redis-cli --cluster create "${IP1}:%d" "${IP2}:%d" "${IP3}:%d" --cluster-replicas 0 --cluster-yes

echo "[redis cluster] cluster info for %s"
"$CONTAINER_ENGINE" exec %s redis-cli -p %d cluster info | grep cluster_state || true
"$CONTAINER_ENGINE" exec %s redis-cli -p %d cluster nodes || true
`, name, node1, node2, node3, name, node1, port1, port2, port3, name, node1, port1, node1, port1)
}

func redisClusterCheckScript(name, node1, node2, node3 string, port1, port2, port3 int) string {
	return fmt.Sprintf(`echo "[redis cluster] container status for %s"
"$CONTAINER_ENGINE" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /%s|%s|%s/'

echo "[redis cluster] ping all nodes for %s"
"$CONTAINER_ENGINE" exec %s redis-cli -p %d ping
"$CONTAINER_ENGINE" exec %s redis-cli -p %d ping
"$CONTAINER_ENGINE" exec %s redis-cli -p %d ping

echo "[redis cluster] cluster state for %s"
OK=0
FINAL_INFO=""
for _ in $(seq 1 10); do
  FINAL_INFO="$("$CONTAINER_ENGINE" exec %s redis-cli -p %d cluster info)"
  CSTATE="$(echo "$FINAL_INFO" | grep '^cluster_state:' | awk -F: '{print $2}' | tr -d '\r')"
  if [ "$CSTATE" = "ok" ]; then
    OK=1
    break
  fi
  sleep 2
done
echo "$FINAL_INFO" | grep -E 'cluster_state|cluster_known_nodes|cluster_size'
if [ "$OK" -ne 1 ]; then
  echo "$FINAL_INFO" >&2
  exit 1
fi

echo "[redis cluster] cluster nodes for %s"
"$CONTAINER_ENGINE" exec %s redis-cli -p %d cluster nodes
`, name, node1, node2, node3, name, node1, port1, node2, port2, node3, port3, name, node1, port1, name, node1, port1)
}

func redisClusterReadme(name, version, image string, port1, port2, port3 int) string {
	return fmt.Sprintf("# Redis %s\n\n- template: cluster\n- version: %s\n- image: %s\n- node ports: %d, %d, %d\n", name, version, image, port1, port2, port3)
}
