package consul

import (
	"fmt"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

const (
	clusterTemplate        = "cluster"
	defaultConsul1HTTPPort = 8500
	defaultConsul1DNSPort  = 8600
	defaultConsul2HTTPPort = 9500
	defaultConsul3HTTPPort = 10500
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
		return model.BlueprintService{}, fmt.Errorf("normalize consul cluster version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))

	values[runtimecompose.ValueConsul1ServiceName] = defaultStringValue(values[runtimecompose.ValueConsul1ServiceName], name+"-consul1")
	values[runtimecompose.ValueConsul2ServiceName] = defaultStringValue(values[runtimecompose.ValueConsul2ServiceName], name+"-consul2")
	values[runtimecompose.ValueConsul3ServiceName] = defaultStringValue(values[runtimecompose.ValueConsul3ServiceName], name+"-consul3")
	values[runtimecompose.ValueConsul1ContainerName] = defaultStringValue(values[runtimecompose.ValueConsul1ContainerName], values[runtimecompose.ValueConsul1ServiceName].(string))
	values[runtimecompose.ValueConsul2ContainerName] = defaultStringValue(values[runtimecompose.ValueConsul2ContainerName], values[runtimecompose.ValueConsul2ServiceName].(string))
	values[runtimecompose.ValueConsul3ContainerName] = defaultStringValue(values[runtimecompose.ValueConsul3ContainerName], values[runtimecompose.ValueConsul3ServiceName].(string))
	values[runtimecompose.ValueConsul1DataDir] = defaultStringValue(values[runtimecompose.ValueConsul1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueConsul1ServiceName].(string)))
	values[runtimecompose.ValueConsul2DataDir] = defaultStringValue(values[runtimecompose.ValueConsul2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueConsul2ServiceName].(string)))
	values[runtimecompose.ValueConsul3DataDir] = defaultStringValue(values[runtimecompose.ValueConsul3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueConsul3ServiceName].(string)))

	var err error
	values[runtimecompose.ValueConsul1HTTPPort], err = allocateOrReservePort(values[runtimecompose.ValueConsul1HTTPPort], hasValue(input.Values, runtimecompose.ValueConsul1HTTPPort), defaultConsul1HTTPPort, "consul cluster consul1_http_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueConsul2HTTPPort], err = allocateOrReservePort(values[runtimecompose.ValueConsul2HTTPPort], hasValue(input.Values, runtimecompose.ValueConsul2HTTPPort), defaultConsul2HTTPPort, "consul cluster consul2_http_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueConsul3HTTPPort], err = allocateOrReservePort(values[runtimecompose.ValueConsul3HTTPPort], hasValue(input.Values, runtimecompose.ValueConsul3HTTPPort), defaultConsul3HTTPPort, "consul cluster consul3_http_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueConsul1DNSPort], err = allocateOrReservePort(values[runtimecompose.ValueConsul1DNSPort], hasValue(input.Values, runtimecompose.ValueConsul1DNSPort), defaultConsul1DNSPort, "consul cluster consul1_dns_port")
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
			dnsPortKey   string
			nodeName     string
			withUI       bool
		}
		nodes := []nodeSpec{
			{runtimecompose.ValueConsul1ServiceName, runtimecompose.ValueConsul1ContainerName, runtimecompose.ValueConsul1DataDir, runtimecompose.ValueConsul1HTTPPort, runtimecompose.ValueConsul1DNSPort, "consul1", true},
			{runtimecompose.ValueConsul2ServiceName, runtimecompose.ValueConsul2ContainerName, runtimecompose.ValueConsul2DataDir, runtimecompose.ValueConsul2HTTPPort, "", "consul2", false},
			{runtimecompose.ValueConsul3ServiceName, runtimecompose.ValueConsul3ContainerName, runtimecompose.ValueConsul3DataDir, runtimecompose.ValueConsul3HTTPPort, "", "consul3", false},
		}

		clusterContexts := make([]runtime.ComposeContext, 0, len(nodes))
		for _, node := range nodes {
			serviceName := service.Values[node.serviceKey].(string)
			containerName := service.Values[node.containerKey].(string)
			dataDir := service.Values[node.dataDirKey].(string)
			httpPort := service.Values[node.httpPortKey].(int)
			envKeyPrefix := serviceEnvKeyPrefix(serviceName)

			command := []string{
				"agent",
				"-server",
				"-node=" + node.nodeName,
				"-bootstrap-expect=3",
				"-retry-join=consul1",
				"-retry-join=consul2",
				"-retry-join=consul3",
				"-client=0.0.0.0",
				"-bind=0.0.0.0",
				"-data-dir=/consul/data",
			}
			if node.withUI {
				command = []string{
					"agent",
					"-server",
					"-ui",
					"-node=" + node.nodeName,
					"-bootstrap-expect=3",
					"-retry-join=consul1",
					"-retry-join=consul2",
					"-retry-join=consul3",
					"-client=0.0.0.0",
					"-bind=0.0.0.0",
					"-data-dir=/consul/data",
				}
			}

			ports := []runtime.PortBinding{
				{HostPort: httpPort, ContainerPort: 8500, Protocol: "tcp"},
			}
			envContent := fmt.Sprintf("%s_VERSION=%s\n%s_IMAGE=%s\n%s_HTTP_PORT=%d\n", envKeyPrefix, version, envKeyPrefix, image, envKeyPrefix, httpPort)
			if node.dnsPortKey != "" {
				dnsPort := service.Values[node.dnsPortKey].(int)
				ports = append(ports, runtime.PortBinding{HostPort: dnsPort, ContainerPort: 8600, Protocol: "udp"})
				envContent += fmt.Sprintf("%s_DNS_PORT=%d\n", envKeyPrefix, dnsPort)
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
					Command:       command,
					Ports:         ports,
					Volumes:       []runtime.VolumeMount{{Source: dataDir, Target: "/consul/data"}},
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
						Name:      "consul-cluster-env",
						PathKey:   "env_file",
						Content:   envContent,
						Mode:      0o644,
						MergeMode: runtime.AssetMergeEnv,
					},
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			})
		}

		consul1HTTPPort := service.Values[runtimecompose.ValueConsul1HTTPPort].(int)
		consul2HTTPPort := service.Values[runtimecompose.ValueConsul2HTTPPort].(int)
		consul3HTTPPort := service.Values[runtimecompose.ValueConsul3HTTPPort].(int)
		clusterContexts[0] = addClusterSharedAssets(clusterContexts[0], service, version, image, consul1HTTPPort, consul2HTTPPort, consul3HTTPPort)
		for _, contextItem := range clusterContexts {
			contexts = append(contexts, contextItem)
		}
	}
	return contexts, nil
}

func addClusterSharedAssets(context runtime.ComposeContext, service model.BlueprintService, version, image string, consul1HTTPPort, consul2HTTPPort, consul3HTTPPort int) runtime.ComposeContext {
	context.Assets = append(context.Assets,
		runtime.AssetSpec{
			Name:      "consul-cluster-build",
			PathKey:   "build_script",
			Content:   fmt.Sprintf("echo \"consul %s (%s) cluster compose stack started\"\n", service.Name, version),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:    "consul-cluster-check",
			PathKey: "check_script",
			Content: fmt.Sprintf(
				"LEADER=\"$(curl -fsS \"http://127.0.0.1:%d/v1/status/leader\" | tr -d '\\\"')\"\n"+
					"[ -n \"$LEADER\" ]\n"+
					"MEMBERS=\"$(curl -fsS \"http://127.0.0.1:%d/v1/agent/members\" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')\"\n"+
					"[ \"$MEMBERS\" -ge 3 ]\n"+
					"RAFT=\"$(curl -fsS \"http://127.0.0.1:%d/v1/operator/raft/configuration\")\"\n"+
					"echo \"$RAFT\" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(\"raft_servers=\",len(d.get(\"Servers\",[])));'\n"+
					"KEY=\"zygarde/cluster/smoke/$(date +%%s)\"\nVAL=\"ok-$(date +%%s)\"\n"+
					"curl -fsS -X PUT --data \"$VAL\" \"http://127.0.0.1:%d/v1/kv/$KEY\" >/dev/null\n"+
					"OUT=\"$(curl -fsS \"http://127.0.0.1:%d/v1/kv/$KEY?raw\")\"\n[ \"$OUT\" = \"$VAL\" ]\n",
				consul1HTTPPort, consul1HTTPPort, consul1HTTPPort, consul2HTTPPort, consul3HTTPPort,
			),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:      "consul-cluster-readme",
			PathKey:   "readme_file",
			Content:   consulClusterReadme(service, version, image),
			Mode:      0o644,
			MergeMode: runtime.AssetMergeReadme,
		},
	)
	return context
}

func consulClusterReadme(service model.BlueprintService, version, image string) string {
	return fmt.Sprintf(
		"# Consul %s Cluster\n\n- version: %s\n- image: %s\n- consul1 http port: %d\n- consul1 dns port: %d\n- consul2 http port: %d\n- consul3 http port: %d\n",
		service.Name,
		version,
		image,
		service.Values[runtimecompose.ValueConsul1HTTPPort].(int),
		service.Values[runtimecompose.ValueConsul1DNSPort].(int),
		service.Values[runtimecompose.ValueConsul2HTTPPort].(int),
		service.Values[runtimecompose.ValueConsul3HTTPPort].(int),
	)
}

func (*clusterSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template != clusterTemplate {
		return fmt.Errorf("consul cluster validate: unexpected template %q", service.Template)
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("consul cluster validate: unexpected middleware %q", service.Middleware)
	}
	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("consul cluster validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("consul cluster validate version: %w", err)
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueConsul1ServiceName, "consul1_service_name"},
		{runtimecompose.ValueConsul2ServiceName, "consul2_service_name"},
		{runtimecompose.ValueConsul3ServiceName, "consul3_service_name"},
		{runtimecompose.ValueConsul1ContainerName, "consul1_container_name"},
		{runtimecompose.ValueConsul2ContainerName, "consul2_container_name"},
		{runtimecompose.ValueConsul3ContainerName, "consul3_container_name"},
		{runtimecompose.ValueConsul1DataDir, "consul1_data_dir"},
		{runtimecompose.ValueConsul2DataDir, "consul2_data_dir"},
		{runtimecompose.ValueConsul3DataDir, "consul3_data_dir"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("consul cluster validate %s: must be a non-empty string", field.name)
		}
	}
	seen := map[int]string{}
	for port, name := range map[int]string{
		service.Values[runtimecompose.ValueConsul1HTTPPort].(int): "consul1_http_port",
		service.Values[runtimecompose.ValueConsul1DNSPort].(int):  "consul1_dns_port",
		service.Values[runtimecompose.ValueConsul2HTTPPort].(int): "consul2_http_port",
		service.Values[runtimecompose.ValueConsul3HTTPPort].(int): "consul3_http_port",
	} {
		if port <= 0 {
			return fmt.Errorf("consul cluster validate %s: must be greater than 0", name)
		}
		if previous, ok := seen[port]; ok {
			return fmt.Errorf("consul cluster validate ports: %s and %s must be different", previous, name)
		}
		seen[port] = name
	}
	return nil
}

func (*clusterSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:              defaultVersion,
		runtimecompose.ValueImage:                "",
		runtimecompose.ValueConsul1ServiceName:   "",
		runtimecompose.ValueConsul2ServiceName:   "",
		runtimecompose.ValueConsul3ServiceName:   "",
		runtimecompose.ValueConsul1ContainerName: "",
		runtimecompose.ValueConsul2ContainerName: "",
		runtimecompose.ValueConsul3ContainerName: "",
		runtimecompose.ValueConsul1DataDir:       "",
		runtimecompose.ValueConsul2DataDir:       "",
		runtimecompose.ValueConsul3DataDir:       "",
		runtimecompose.ValueConsul1HTTPPort:      defaultConsul1HTTPPort,
		runtimecompose.ValueConsul1DNSPort:       defaultConsul1DNSPort,
		runtimecompose.ValueConsul2HTTPPort:      defaultConsul2HTTPPort,
		runtimecompose.ValueConsul3HTTPPort:      defaultConsul3HTTPPort,
	}
}
