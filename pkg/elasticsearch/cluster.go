package elasticsearch

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
	defaultES1HTTPPort = 9220
	defaultES2HTTPPort = 9221
	defaultES3HTTPPort = 9222
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
		return model.BlueprintService{}, fmt.Errorf("normalize elasticsearch cluster version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))

	values[runtimecompose.ValueES1ServiceName] = defaultStringValue(values[runtimecompose.ValueES1ServiceName], name+"-es1")
	values[runtimecompose.ValueES2ServiceName] = defaultStringValue(values[runtimecompose.ValueES2ServiceName], name+"-es2")
	values[runtimecompose.ValueES3ServiceName] = defaultStringValue(values[runtimecompose.ValueES3ServiceName], name+"-es3")
	values[runtimecompose.ValueES1ContainerName] = defaultStringValue(values[runtimecompose.ValueES1ContainerName], values[runtimecompose.ValueES1ServiceName].(string))
	values[runtimecompose.ValueES2ContainerName] = defaultStringValue(values[runtimecompose.ValueES2ContainerName], values[runtimecompose.ValueES2ServiceName].(string))
	values[runtimecompose.ValueES3ContainerName] = defaultStringValue(values[runtimecompose.ValueES3ContainerName], values[runtimecompose.ValueES3ServiceName].(string))
	values[runtimecompose.ValueES1DataDir] = defaultStringValue(values[runtimecompose.ValueES1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueES1ServiceName].(string)))
	values[runtimecompose.ValueES2DataDir] = defaultStringValue(values[runtimecompose.ValueES2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueES2ServiceName].(string)))
	values[runtimecompose.ValueES3DataDir] = defaultStringValue(values[runtimecompose.ValueES3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueES3ServiceName].(string)))

	var err error
	values[runtimecompose.ValueES1HTTPPort], err = allocateOrReservePort(values[runtimecompose.ValueES1HTTPPort], hasValue(input.Values, runtimecompose.ValueES1HTTPPort), defaultES1HTTPPort, "elasticsearch cluster es1_http_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueES2HTTPPort], err = allocateOrReservePort(values[runtimecompose.ValueES2HTTPPort], hasValue(input.Values, runtimecompose.ValueES2HTTPPort), defaultES2HTTPPort, "elasticsearch cluster es2_http_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueES3HTTPPort], err = allocateOrReservePort(values[runtimecompose.ValueES3HTTPPort], hasValue(input.Values, runtimecompose.ValueES3HTTPPort), defaultES3HTTPPort, "elasticsearch cluster es3_http_port")
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
		serviceNames := []string{
			service.Values[runtimecompose.ValueES1ServiceName].(string),
			service.Values[runtimecompose.ValueES2ServiceName].(string),
			service.Values[runtimecompose.ValueES3ServiceName].(string),
		}
		clusterName := "zygarde-es"
		seedHosts := strings.Join(serviceNames, ",")

		type nodeSpec struct {
			nodeName     string
			serviceKey   string
			containerKey string
			dataDirKey   string
			httpPortKey  string
		}
		nodes := []nodeSpec{
			{"es1", runtimecompose.ValueES1ServiceName, runtimecompose.ValueES1ContainerName, runtimecompose.ValueES1DataDir, runtimecompose.ValueES1HTTPPort},
			{"es2", runtimecompose.ValueES2ServiceName, runtimecompose.ValueES2ContainerName, runtimecompose.ValueES2DataDir, runtimecompose.ValueES2HTTPPort},
			{"es3", runtimecompose.ValueES3ServiceName, runtimecompose.ValueES3ContainerName, runtimecompose.ValueES3DataDir, runtimecompose.ValueES3HTTPPort},
		}

		clusterContexts := make([]runtime.ComposeContext, 0, len(nodes))
		for _, node := range nodes {
			serviceName := service.Values[node.serviceKey].(string)
			containerName := service.Values[node.containerKey].(string)
			dataDir := service.Values[node.dataDirKey].(string)
			httpPort := service.Values[node.httpPortKey].(int)
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
						"node.name":                    node.nodeName,
						"cluster.name":                 clusterName,
						"discovery.seed_hosts":         seedHosts,
						"cluster.initial_master_nodes": "es1,es2,es3",
						"xpack.security.enabled":       "false",
						"ES_JAVA_OPTS":                 "-Xms512m -Xmx512m",
					},
					Ports: []runtime.PortBinding{
						{HostPort: httpPort, ContainerPort: 9200, Protocol: "tcp"},
					},
					Volumes: []runtime.VolumeMount{
						{Source: dataDir, Target: defaultDataDirTarget},
					},
					HealthCheck: &runtime.HealthCheck{
						Test:        []string{"CMD-SHELL", "curl -fsS http://127.0.0.1:9200/_cluster/health >/dev/null 2>&1"},
						Interval:    "5s",
						Timeout:     "5s",
						Retries:     60,
						StartPeriod: "30s",
					},
				},
				Assets: []runtime.AssetSpec{
					{
						Name:      "elasticsearch-cluster-env",
						PathKey:   "env_file",
						Content:   fmt.Sprintf("%s_VERSION=%s\n%s_HTTP_PORT=%d\n", envKeyPrefix, version, envKeyPrefix, httpPort),
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
	es1Port := service.Values[runtimecompose.ValueES1HTTPPort].(int)
	es2Port := service.Values[runtimecompose.ValueES2HTTPPort].(int)
	es3Port := service.Values[runtimecompose.ValueES3HTTPPort].(int)
	context.Assets = append(context.Assets,
		runtime.AssetSpec{
			Name:      "elasticsearch-cluster-build",
			PathKey:   "build_script",
			Content:   fmt.Sprintf("echo \"Elasticsearch %s (%s) cluster compose stack started\"\n", service.Name, version),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:    "elasticsearch-cluster-check",
			PathKey: "check_script",
			Content: fmt.Sprintf(
				"HEALTH=\"$(curl -fsS http://127.0.0.1:%d/_cluster/health)\"\n"+
					"echo \"$HEALTH\"\n"+
					"NODES=\"$(echo \"$HEALTH\" | python3 -c 'import json,sys; print(json.load(sys.stdin).get(\"number_of_nodes\",0))')\"\n"+
					"[ \"$NODES\" -ge 3 ]\n"+
					"curl -fsS http://127.0.0.1:%d/_cat/nodes?v\n"+
					"IDX=\"zygarde-cluster-smoke\"\n"+
					"curl -fsS -X PUT http://127.0.0.1:%d/${IDX} >/dev/null\n"+
					"curl -fsS -X POST http://127.0.0.1:%d/${IDX}/_doc/1 -H 'Content-Type: application/json' -d '{\"msg\":\"ok\"}' >/dev/null\n"+
					"curl -fsS http://127.0.0.1:%d/${IDX}/_doc/1 | python3 -c 'import json,sys; d=json.load(sys.stdin); v=d.get(\"_source\",{}).get(\"msg\"); assert v==\"ok\", v; print(\"doc=ok\")'\n",
				es1Port,
				es1Port,
				es2Port,
				es3Port,
				es1Port,
			),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:      "elasticsearch-cluster-readme",
			PathKey:   "readme_file",
			Content:   elasticsearchClusterReadme(service, version, image),
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
		return fmt.Errorf("elasticsearch cluster validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != clusterTemplate {
		return fmt.Errorf("elasticsearch cluster validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || strings.TrimSpace(version) == "" {
		return fmt.Errorf("elasticsearch cluster validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("elasticsearch cluster validate version: %w", err)
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueES1ServiceName, "es1_service_name"},
		{runtimecompose.ValueES2ServiceName, "es2_service_name"},
		{runtimecompose.ValueES3ServiceName, "es3_service_name"},
		{runtimecompose.ValueES1ContainerName, "es1_container_name"},
		{runtimecompose.ValueES2ContainerName, "es2_container_name"},
		{runtimecompose.ValueES3ContainerName, "es3_container_name"},
		{runtimecompose.ValueES1DataDir, "es1_data_dir"},
		{runtimecompose.ValueES2DataDir, "es2_data_dir"},
		{runtimecompose.ValueES3DataDir, "es3_data_dir"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("elasticsearch cluster validate %s: must be a non-empty string", field.name)
		}
	}

	seen := map[int]string{}
	for _, field := range []struct {
		key  string
		name string
	}{
		{runtimecompose.ValueES1HTTPPort, "es1_http_port"},
		{runtimecompose.ValueES2HTTPPort, "es2_http_port"},
		{runtimecompose.ValueES3HTTPPort, "es3_http_port"},
	} {
		port, err := normalizePort(service.Values[field.key])
		if err != nil {
			return fmt.Errorf("elasticsearch cluster validate %s: %w", field.name, err)
		}
		if port <= 0 {
			return fmt.Errorf("elasticsearch cluster validate %s: must be greater than 0", field.name)
		}
		if previous, ok := seen[port]; ok {
			return fmt.Errorf("elasticsearch cluster validate ports: %s and %s must be different", previous, field.name)
		}
		seen[port] = field.name
	}

	return nil
}

func (*clusterSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:     defaultVersion,
		runtimecompose.ValueImage:       "",
		runtimecompose.ValueES1DataDir:  "",
		runtimecompose.ValueES2DataDir:  "",
		runtimecompose.ValueES3DataDir:  "",
		runtimecompose.ValueES1HTTPPort: defaultES1HTTPPort,
		runtimecompose.ValueES2HTTPPort: defaultES2HTTPPort,
		runtimecompose.ValueES3HTTPPort: defaultES3HTTPPort,
	}
}

func elasticsearchClusterReadme(service model.BlueprintService, version, image string) string {
	return fmt.Sprintf(
		"# Elasticsearch %s Cluster\n\n- version: %s\n- image: %s\n- es1 http port: %d\n- es2 http port: %d\n- es3 http port: %d\n",
		service.Name,
		version,
		image,
		service.Values[runtimecompose.ValueES1HTTPPort].(int),
		service.Values[runtimecompose.ValueES2HTTPPort].(int),
		service.Values[runtimecompose.ValueES3HTTPPort].(int),
	)
}
