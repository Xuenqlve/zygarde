package etcd

import (
	"fmt"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

const (
	clusterTemplate         = "cluster"
	defaultEtcd1ClientPort  = 2379
	defaultEtcd2ClientPort  = 2479
	defaultEtcd3ClientPort  = 2579
	defaultClusterTokenName = "zygarde-etcd-cluster"
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
		return model.BlueprintService{}, fmt.Errorf("normalize etcd cluster version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))
	values[runtimecompose.ValueClusterToken] = defaultStringValue(values[runtimecompose.ValueClusterToken], defaultClusterTokenName)

	values[runtimecompose.ValueEtcd1ServiceName] = defaultStringValue(values[runtimecompose.ValueEtcd1ServiceName], name+"-etcd1")
	values[runtimecompose.ValueEtcd2ServiceName] = defaultStringValue(values[runtimecompose.ValueEtcd2ServiceName], name+"-etcd2")
	values[runtimecompose.ValueEtcd3ServiceName] = defaultStringValue(values[runtimecompose.ValueEtcd3ServiceName], name+"-etcd3")
	values[runtimecompose.ValueEtcd1ContainerName] = defaultStringValue(values[runtimecompose.ValueEtcd1ContainerName], values[runtimecompose.ValueEtcd1ServiceName].(string))
	values[runtimecompose.ValueEtcd2ContainerName] = defaultStringValue(values[runtimecompose.ValueEtcd2ContainerName], values[runtimecompose.ValueEtcd2ServiceName].(string))
	values[runtimecompose.ValueEtcd3ContainerName] = defaultStringValue(values[runtimecompose.ValueEtcd3ContainerName], values[runtimecompose.ValueEtcd3ServiceName].(string))
	values[runtimecompose.ValueEtcd1DataDir] = defaultStringValue(values[runtimecompose.ValueEtcd1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueEtcd1ServiceName].(string)))
	values[runtimecompose.ValueEtcd2DataDir] = defaultStringValue(values[runtimecompose.ValueEtcd2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueEtcd2ServiceName].(string)))
	values[runtimecompose.ValueEtcd3DataDir] = defaultStringValue(values[runtimecompose.ValueEtcd3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueEtcd3ServiceName].(string)))

	var err error
	values[runtimecompose.ValueEtcd1ClientPort], err = allocateOrReservePort(values[runtimecompose.ValueEtcd1ClientPort], hasValue(input.Values, runtimecompose.ValueEtcd1ClientPort), defaultEtcd1ClientPort, "etcd cluster etcd1_client_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueEtcd2ClientPort], err = allocateOrReservePort(values[runtimecompose.ValueEtcd2ClientPort], hasValue(input.Values, runtimecompose.ValueEtcd2ClientPort), defaultEtcd2ClientPort, "etcd cluster etcd2_client_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueEtcd3ClientPort], err = allocateOrReservePort(values[runtimecompose.ValueEtcd3ClientPort], hasValue(input.Values, runtimecompose.ValueEtcd3ClientPort), defaultEtcd3ClientPort, "etcd cluster etcd3_client_port")
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
		clusterToken := service.Values[runtimecompose.ValueClusterToken].(string)

		type nodeSpec struct {
			serviceKey    string
			containerKey  string
			dataDirKey    string
			clientPortKey string
			name          string
		}
		nodes := []nodeSpec{
			{runtimecompose.ValueEtcd1ServiceName, runtimecompose.ValueEtcd1ContainerName, runtimecompose.ValueEtcd1DataDir, runtimecompose.ValueEtcd1ClientPort, "etcd1"},
			{runtimecompose.ValueEtcd2ServiceName, runtimecompose.ValueEtcd2ContainerName, runtimecompose.ValueEtcd2DataDir, runtimecompose.ValueEtcd2ClientPort, "etcd2"},
			{runtimecompose.ValueEtcd3ServiceName, runtimecompose.ValueEtcd3ContainerName, runtimecompose.ValueEtcd3DataDir, runtimecompose.ValueEtcd3ClientPort, "etcd3"},
		}

		clusterDef := "etcd1=http://etcd1:2380,etcd2=http://etcd2:2380,etcd3=http://etcd3:2380"
		clusterContexts := make([]runtime.ComposeContext, 0, len(nodes))
		for _, node := range nodes {
			serviceName := service.Values[node.serviceKey].(string)
			containerName := service.Values[node.containerKey].(string)
			dataDir := service.Values[node.dataDirKey].(string)
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
						"ALLOW_NONE_AUTHENTICATION":        "yes",
						"ETCD_NAME":                        node.name,
						"ETCD_DATA_DIR":                    "/etcd-data",
						"ETCD_LISTEN_CLIENT_URLS":          "http://0.0.0.0:2379",
						"ETCD_ADVERTISE_CLIENT_URLS":       fmt.Sprintf("http://%s:2379", node.name),
						"ETCD_LISTEN_PEER_URLS":            "http://0.0.0.0:2380",
						"ETCD_INITIAL_ADVERTISE_PEER_URLS": fmt.Sprintf("http://%s:2380", node.name),
						"ETCD_INITIAL_CLUSTER":             clusterDef,
						"ETCD_INITIAL_CLUSTER_STATE":       "new",
						"ETCD_INITIAL_CLUSTER_TOKEN":       clusterToken,
					},
					Ports: []runtime.PortBinding{
						{HostPort: clientPort, ContainerPort: 2379, Protocol: "tcp"},
					},
					Volumes: []runtime.VolumeMount{
						{Source: dataDir, Target: "/etcd-data"},
					},
					HealthCheck: &runtime.HealthCheck{
						Test:        []string{"CMD-SHELL", fmt.Sprintf("etcdctl --endpoints=http://%s:2379,http://etcd2:2379,http://etcd3:2379 endpoint health >/dev/null 2>&1", node.name)},
						Interval:    "5s",
						Timeout:     "5s",
						Retries:     60,
						StartPeriod: "10s",
					},
				},
				Assets: []runtime.AssetSpec{
					{
						Name:    "etcd-cluster-env",
						PathKey: "env_file",
						Content: fmt.Sprintf(
							"%s_VERSION=%s\n%s_IMAGE=%s\n%s_CLIENT_PORT=%d\n%s_CLUSTER_TOKEN=%s\n",
							envKeyPrefix, version,
							envKeyPrefix, image,
							envKeyPrefix, clientPort,
							envKeyPrefix, clusterToken,
						),
						Mode:      0o644,
						MergeMode: runtime.AssetMergeEnv,
					},
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			})
		}

		container1 := service.Values[runtimecompose.ValueEtcd1ContainerName].(string)
		container2 := service.Values[runtimecompose.ValueEtcd2ContainerName].(string)
		container3 := service.Values[runtimecompose.ValueEtcd3ContainerName].(string)
		clusterContexts[0] = addClusterSharedAssets(clusterContexts[0], service, version, image, container1, container2, container3, clusterToken)
		for _, contextItem := range clusterContexts {
			contexts = append(contexts, contextItem)
		}
	}
	return contexts, nil
}

func addClusterSharedAssets(context runtime.ComposeContext, service model.BlueprintService, version, image, container1, container2, container3, clusterToken string) runtime.ComposeContext {
	context.Assets = append(context.Assets,
		runtime.AssetSpec{
			Name:      "etcd-cluster-build",
			PathKey:   "build_script",
			Content:   fmt.Sprintf("echo \"etcd %s (%s) cluster compose stack started\"\n", service.Name, version),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:    "etcd-cluster-check",
			PathKey: "check_script",
			Content: fmt.Sprintf(
				"ENDPOINTS=\"http://etcd1:2379,http://etcd2:2379,http://etcd3:2379\"\n"+
					"\"$CONTAINER_ENGINE\" exec %s etcdctl --endpoints=\"$ENDPOINTS\" endpoint health\n"+
					"MEMBERS=\"$(\"$CONTAINER_ENGINE\" exec %s etcdctl --endpoints=\"$ENDPOINTS\" member list | tee /dev/stderr | wc -l | tr -d '[:space:]')\"\n"+
					"[ \"$MEMBERS\" -ge 3 ]\n"+
					"KEY=\"zygarde-cluster-smoke-$(date +%%s)\"\n"+
					"VAL=\"ok-$(date +%%s)\"\n"+
					"\"$CONTAINER_ENGINE\" exec %s etcdctl --endpoints=\"$ENDPOINTS\" put \"$KEY\" \"$VAL\" >/dev/null\n"+
					"OUT=\"$(\"$CONTAINER_ENGINE\" exec %s etcdctl --endpoints=\"$ENDPOINTS\" get \"$KEY\" --print-value-only | tr -d '\\r')\"\n"+
					"[ \"$OUT\" = \"$VAL\" ]\n",
				container1, container1, container2, container3,
			),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:      "etcd-cluster-readme",
			PathKey:   "readme_file",
			Content:   etcdClusterReadme(service, version, image, clusterToken),
			Mode:      0o644,
			MergeMode: runtime.AssetMergeReadme,
		},
	)
	return context
}

func etcdClusterReadme(service model.BlueprintService, version, image, clusterToken string) string {
	return fmt.Sprintf(
		"# etcd %s Cluster\n\n- version: %s\n- image: %s\n- cluster token: %s\n- etcd1 client port: %d\n- etcd2 client port: %d\n- etcd3 client port: %d\n",
		service.Name,
		version,
		image,
		clusterToken,
		service.Values[runtimecompose.ValueEtcd1ClientPort].(int),
		service.Values[runtimecompose.ValueEtcd2ClientPort].(int),
		service.Values[runtimecompose.ValueEtcd3ClientPort].(int),
	)
}

func (*clusterSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template != clusterTemplate {
		return fmt.Errorf("etcd cluster validate: unexpected template %q", service.Template)
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("etcd cluster validate: unexpected middleware %q", service.Middleware)
	}
	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("etcd cluster validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("etcd cluster validate version: %w", err)
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueClusterToken, "cluster_token"},
		{runtimecompose.ValueEtcd1ServiceName, "etcd1_service_name"},
		{runtimecompose.ValueEtcd2ServiceName, "etcd2_service_name"},
		{runtimecompose.ValueEtcd3ServiceName, "etcd3_service_name"},
		{runtimecompose.ValueEtcd1ContainerName, "etcd1_container_name"},
		{runtimecompose.ValueEtcd2ContainerName, "etcd2_container_name"},
		{runtimecompose.ValueEtcd3ContainerName, "etcd3_container_name"},
		{runtimecompose.ValueEtcd1DataDir, "etcd1_data_dir"},
		{runtimecompose.ValueEtcd2DataDir, "etcd2_data_dir"},
		{runtimecompose.ValueEtcd3DataDir, "etcd3_data_dir"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("etcd cluster validate %s: must be a non-empty string", field.name)
		}
	}
	portKeys := []struct{ key, name string }{
		{runtimecompose.ValueEtcd1ClientPort, "etcd1_client_port"},
		{runtimecompose.ValueEtcd2ClientPort, "etcd2_client_port"},
		{runtimecompose.ValueEtcd3ClientPort, "etcd3_client_port"},
	}
	seen := make(map[int]string, len(portKeys))
	for _, field := range portKeys {
		port, err := normalizePort(service.Values[field.key])
		if err != nil {
			return fmt.Errorf("etcd cluster validate %s: %w", field.name, err)
		}
		if port <= 0 {
			return fmt.Errorf("etcd cluster validate %s: must be greater than 0", field.name)
		}
		if previous, ok := seen[port]; ok {
			return fmt.Errorf("etcd cluster validate ports: %s and %s must be different", previous, field.name)
		}
		seen[port] = field.name
	}
	return nil
}

func (*clusterSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:            defaultVersion,
		runtimecompose.ValueImage:              "",
		runtimecompose.ValueClusterToken:       defaultClusterTokenName,
		runtimecompose.ValueEtcd1ServiceName:   "",
		runtimecompose.ValueEtcd2ServiceName:   "",
		runtimecompose.ValueEtcd3ServiceName:   "",
		runtimecompose.ValueEtcd1ContainerName: "",
		runtimecompose.ValueEtcd2ContainerName: "",
		runtimecompose.ValueEtcd3ContainerName: "",
		runtimecompose.ValueEtcd1DataDir:       "",
		runtimecompose.ValueEtcd2DataDir:       "",
		runtimecompose.ValueEtcd3DataDir:       "",
		runtimecompose.ValueEtcd1ClientPort:    defaultEtcd1ClientPort,
		runtimecompose.ValueEtcd2ClientPort:    defaultEtcd2ClientPort,
		runtimecompose.ValueEtcd3ClientPort:    defaultEtcd3ClientPort,
	}
}
