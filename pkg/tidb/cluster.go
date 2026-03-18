package tidb

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
	defaultPD1Port         = 2379
	defaultPD2Port         = 2479
	defaultPD3Port         = 2579
	defaultTiDB1Port       = 4000
	defaultTiDB2Port       = 4001
	defaultTiDB1StatusPort = 10080
	defaultTiDB2StatusPort = 10081
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
		return model.BlueprintService{}, fmt.Errorf("normalize tidb cluster version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version

	values[runtimecompose.ValuePDImage] = defaultStringValue(values[runtimecompose.ValuePDImage], pdImageForVersion(version))
	values[runtimecompose.ValueTiKVImage] = defaultStringValue(values[runtimecompose.ValueTiKVImage], tikvImageForVersion(version))
	values[runtimecompose.ValueTiDBImage] = defaultStringValue(values[runtimecompose.ValueTiDBImage], tidbImageForVersion(version))

	values[runtimecompose.ValuePD1ServiceName] = defaultStringValue(values[runtimecompose.ValuePD1ServiceName], name+"-pd1")
	values[runtimecompose.ValuePD2ServiceName] = defaultStringValue(values[runtimecompose.ValuePD2ServiceName], name+"-pd2")
	values[runtimecompose.ValuePD3ServiceName] = defaultStringValue(values[runtimecompose.ValuePD3ServiceName], name+"-pd3")
	values[runtimecompose.ValueTiKV1ServiceName] = defaultStringValue(values[runtimecompose.ValueTiKV1ServiceName], name+"-tikv1")
	values[runtimecompose.ValueTiKV2ServiceName] = defaultStringValue(values[runtimecompose.ValueTiKV2ServiceName], name+"-tikv2")
	values[runtimecompose.ValueTiKV3ServiceName] = defaultStringValue(values[runtimecompose.ValueTiKV3ServiceName], name+"-tikv3")
	values[runtimecompose.ValueTiDB1ServiceName] = defaultStringValue(values[runtimecompose.ValueTiDB1ServiceName], name+"-tidb1")
	values[runtimecompose.ValueTiDB2ServiceName] = defaultStringValue(values[runtimecompose.ValueTiDB2ServiceName], name+"-tidb2")

	values[runtimecompose.ValuePD1ContainerName] = defaultStringValue(values[runtimecompose.ValuePD1ContainerName], values[runtimecompose.ValuePD1ServiceName].(string))
	values[runtimecompose.ValuePD2ContainerName] = defaultStringValue(values[runtimecompose.ValuePD2ContainerName], values[runtimecompose.ValuePD2ServiceName].(string))
	values[runtimecompose.ValuePD3ContainerName] = defaultStringValue(values[runtimecompose.ValuePD3ContainerName], values[runtimecompose.ValuePD3ServiceName].(string))
	values[runtimecompose.ValueTiKV1ContainerName] = defaultStringValue(values[runtimecompose.ValueTiKV1ContainerName], values[runtimecompose.ValueTiKV1ServiceName].(string))
	values[runtimecompose.ValueTiKV2ContainerName] = defaultStringValue(values[runtimecompose.ValueTiKV2ContainerName], values[runtimecompose.ValueTiKV2ServiceName].(string))
	values[runtimecompose.ValueTiKV3ContainerName] = defaultStringValue(values[runtimecompose.ValueTiKV3ContainerName], values[runtimecompose.ValueTiKV3ServiceName].(string))
	values[runtimecompose.ValueTiDB1ContainerName] = defaultStringValue(values[runtimecompose.ValueTiDB1ContainerName], values[runtimecompose.ValueTiDB1ServiceName].(string))
	values[runtimecompose.ValueTiDB2ContainerName] = defaultStringValue(values[runtimecompose.ValueTiDB2ContainerName], values[runtimecompose.ValueTiDB2ServiceName].(string))

	values[runtimecompose.ValuePD1DataDir] = defaultStringValue(values[runtimecompose.ValuePD1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValuePD1ServiceName].(string)))
	values[runtimecompose.ValuePD2DataDir] = defaultStringValue(values[runtimecompose.ValuePD2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValuePD2ServiceName].(string)))
	values[runtimecompose.ValuePD3DataDir] = defaultStringValue(values[runtimecompose.ValuePD3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValuePD3ServiceName].(string)))
	values[runtimecompose.ValueTiKV1DataDir] = defaultStringValue(values[runtimecompose.ValueTiKV1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueTiKV1ServiceName].(string)))
	values[runtimecompose.ValueTiKV2DataDir] = defaultStringValue(values[runtimecompose.ValueTiKV2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueTiKV2ServiceName].(string)))
	values[runtimecompose.ValueTiKV3DataDir] = defaultStringValue(values[runtimecompose.ValueTiKV3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueTiKV3ServiceName].(string)))

	var err error
	values[runtimecompose.ValuePD1Port], err = allocateOrReservePort(values[runtimecompose.ValuePD1Port], hasValue(input.Values, runtimecompose.ValuePD1Port), defaultPD1Port, "tidb cluster pd1_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValuePD2Port], err = allocateOrReservePort(values[runtimecompose.ValuePD2Port], hasValue(input.Values, runtimecompose.ValuePD2Port), defaultPD2Port, "tidb cluster pd2_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValuePD3Port], err = allocateOrReservePort(values[runtimecompose.ValuePD3Port], hasValue(input.Values, runtimecompose.ValuePD3Port), defaultPD3Port, "tidb cluster pd3_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueTiDB1Port], err = allocateOrReservePort(values[runtimecompose.ValueTiDB1Port], hasValue(input.Values, runtimecompose.ValueTiDB1Port), defaultTiDB1Port, "tidb cluster tidb1_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueTiDB2Port], err = allocateOrReservePort(values[runtimecompose.ValueTiDB2Port], hasValue(input.Values, runtimecompose.ValueTiDB2Port), defaultTiDB2Port, "tidb cluster tidb2_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueTiDB1StatusPort], err = allocateOrReservePort(values[runtimecompose.ValueTiDB1StatusPort], hasValue(input.Values, runtimecompose.ValueTiDB1StatusPort), defaultTiDB1StatusPort, "tidb cluster tidb1_status_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueTiDB2StatusPort], err = allocateOrReservePort(values[runtimecompose.ValueTiDB2StatusPort], hasValue(input.Values, runtimecompose.ValueTiDB2StatusPort), defaultTiDB2StatusPort, "tidb cluster tidb2_status_port")
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

	contexts := make([]runtime.EnvironmentContext, 0, len(services)*8)
	for _, service := range services {
		if err := s.Validate(service); err != nil {
			return nil, err
		}

		version := service.Values[runtimecompose.ValueVersion].(string)
		pdImage := service.Values[runtimecompose.ValuePDImage].(string)
		tikvImage := service.Values[runtimecompose.ValueTiKVImage].(string)
		tidbImage := service.Values[runtimecompose.ValueTiDBImage].(string)

		pd1ServiceName := service.Values[runtimecompose.ValuePD1ServiceName].(string)
		pd2ServiceName := service.Values[runtimecompose.ValuePD2ServiceName].(string)
		pd3ServiceName := service.Values[runtimecompose.ValuePD3ServiceName].(string)
		tikv1ServiceName := service.Values[runtimecompose.ValueTiKV1ServiceName].(string)
		tikv2ServiceName := service.Values[runtimecompose.ValueTiKV2ServiceName].(string)
		tikv3ServiceName := service.Values[runtimecompose.ValueTiKV3ServiceName].(string)
		tidb1ServiceName := service.Values[runtimecompose.ValueTiDB1ServiceName].(string)
		tidb2ServiceName := service.Values[runtimecompose.ValueTiDB2ServiceName].(string)

		pdNodes := []struct {
			serviceName   string
			containerName string
			dataDir       string
			port          int
			command       []string
		}{
			{
				serviceName:   pd1ServiceName,
				containerName: service.Values[runtimecompose.ValuePD1ContainerName].(string),
				dataDir:       service.Values[runtimecompose.ValuePD1DataDir].(string),
				port:          service.Values[runtimecompose.ValuePD1Port].(int),
				command: []string{
					"--name=pd1",
					"--data-dir=/data/pd",
					"--client-urls=http://0.0.0.0:2379",
					"--peer-urls=http://0.0.0.0:2380",
					fmt.Sprintf("--advertise-client-urls=http://%s:2379", pd1ServiceName),
					fmt.Sprintf("--advertise-peer-urls=http://%s:2380", pd1ServiceName),
					fmt.Sprintf("--initial-cluster=pd1=http://%s:2380,pd2=http://%s:2380,pd3=http://%s:2380", pd1ServiceName, pd2ServiceName, pd3ServiceName),
					"--force-new-cluster",
				},
			},
			{
				serviceName:   pd2ServiceName,
				containerName: service.Values[runtimecompose.ValuePD2ContainerName].(string),
				dataDir:       service.Values[runtimecompose.ValuePD2DataDir].(string),
				port:          service.Values[runtimecompose.ValuePD2Port].(int),
				command: []string{
					"--name=pd2",
					"--data-dir=/data/pd",
					"--client-urls=http://0.0.0.0:2379",
					"--peer-urls=http://0.0.0.0:2380",
					fmt.Sprintf("--advertise-client-urls=http://%s:2379", pd2ServiceName),
					fmt.Sprintf("--advertise-peer-urls=http://%s:2380", pd2ServiceName),
					fmt.Sprintf("--join=%s:2379", pd1ServiceName),
				},
			},
			{
				serviceName:   pd3ServiceName,
				containerName: service.Values[runtimecompose.ValuePD3ContainerName].(string),
				dataDir:       service.Values[runtimecompose.ValuePD3DataDir].(string),
				port:          service.Values[runtimecompose.ValuePD3Port].(int),
				command: []string{
					"--name=pd3",
					"--data-dir=/data/pd",
					"--client-urls=http://0.0.0.0:2379",
					"--peer-urls=http://0.0.0.0:2380",
					fmt.Sprintf("--advertise-client-urls=http://%s:2379", pd3ServiceName),
					fmt.Sprintf("--advertise-peer-urls=http://%s:2380", pd3ServiceName),
					fmt.Sprintf("--join=%s:2379", pd1ServiceName),
				},
			},
		}

		pdList := fmt.Sprintf("%s:2379,%s:2379,%s:2379", pd1ServiceName, pd2ServiceName, pd3ServiceName)
		tikvNodes := []struct {
			serviceName   string
			containerName string
			dataDir       string
		}{
			{tikv1ServiceName, service.Values[runtimecompose.ValueTiKV1ContainerName].(string), service.Values[runtimecompose.ValueTiKV1DataDir].(string)},
			{tikv2ServiceName, service.Values[runtimecompose.ValueTiKV2ContainerName].(string), service.Values[runtimecompose.ValueTiKV2DataDir].(string)},
			{tikv3ServiceName, service.Values[runtimecompose.ValueTiKV3ContainerName].(string), service.Values[runtimecompose.ValueTiKV3DataDir].(string)},
		}
		tidbNodes := []struct {
			serviceName   string
			containerName string
			port          int
			statusPort    int
		}{
			{tidb1ServiceName, service.Values[runtimecompose.ValueTiDB1ContainerName].(string), service.Values[runtimecompose.ValueTiDB1Port].(int), service.Values[runtimecompose.ValueTiDB1StatusPort].(int)},
			{tidb2ServiceName, service.Values[runtimecompose.ValueTiDB2ContainerName].(string), service.Values[runtimecompose.ValueTiDB2Port].(int), service.Values[runtimecompose.ValueTiDB2StatusPort].(int)},
		}

		clusterContexts := make([]runtime.ComposeContext, 0, 8)
		for _, pdNode := range pdNodes {
			clusterContexts = append(clusterContexts, runtime.ComposeContext{
				EnvType:     runtimeType,
				ServiceName: pdNode.serviceName,
				Middleware:  service.Middleware,
				Template:    service.Template,
				Service: runtime.ServiceSpec{
					Image:         pdImage,
					ContainerName: pdNode.containerName,
					Restart:       "unless-stopped",
					Command:       pdNode.command,
					Ports:         []runtime.PortBinding{{HostPort: pdNode.port, ContainerPort: 2379, Protocol: "tcp"}},
					Volumes:       []runtime.VolumeMount{{Source: pdNode.dataDir, Target: "/data/pd"}},
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			})
		}
		for _, tikvNode := range tikvNodes {
			clusterContexts = append(clusterContexts, runtime.ComposeContext{
				EnvType:     runtimeType,
				ServiceName: tikvNode.serviceName,
				Middleware:  service.Middleware,
				Template:    service.Template,
				Service: runtime.ServiceSpec{
					Image:         tikvImage,
					ContainerName: tikvNode.containerName,
					Restart:       "unless-stopped",
					Command: []string{
						fmt.Sprintf("--pd=%s", pdList),
						"--addr=0.0.0.0:20160",
						fmt.Sprintf("--advertise-addr=%s:20160", tikvNode.serviceName),
						"--data-dir=/data/tikv",
					},
					Volumes: []runtime.VolumeMount{{Source: tikvNode.dataDir, Target: "/data/tikv"}},
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			})
		}
		for _, tidbNode := range tidbNodes {
			clusterContexts = append(clusterContexts, runtime.ComposeContext{
				EnvType:     runtimeType,
				ServiceName: tidbNode.serviceName,
				Middleware:  service.Middleware,
				Template:    service.Template,
				Service: runtime.ServiceSpec{
					Image:         tidbImage,
					ContainerName: tidbNode.containerName,
					Restart:       "unless-stopped",
					Command: []string{
						"--store=tikv",
						fmt.Sprintf("--path=%s", pdList),
						"--host=0.0.0.0",
						"--status=10080",
						fmt.Sprintf("--advertise-address=%s", tidbNode.serviceName),
					},
					Ports: []runtime.PortBinding{
						{HostPort: tidbNode.port, ContainerPort: 4000, Protocol: "tcp"},
						{HostPort: tidbNode.statusPort, ContainerPort: 10080, Protocol: "tcp"},
					},
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			})
		}

		clusterContexts[0] = addClusterSharedAssets(clusterContexts[0], service, version, pdImage, tikvImage, tidbImage)
		for _, contextItem := range clusterContexts {
			contexts = append(contexts, contextItem)
		}
	}
	return contexts, nil
}

func addClusterSharedAssets(context runtime.ComposeContext, service model.BlueprintService, version, pdImage, tikvImage, tidbImage string) runtime.ComposeContext {
	pd1Port := service.Values[runtimecompose.ValuePD1Port].(int)
	tidb1Port := service.Values[runtimecompose.ValueTiDB1Port].(int)
	tidb2Port := service.Values[runtimecompose.ValueTiDB2Port].(int)
	tidb1StatusPort := service.Values[runtimecompose.ValueTiDB1StatusPort].(int)
	tidb2StatusPort := service.Values[runtimecompose.ValueTiDB2StatusPort].(int)

	context.Assets = append(context.Assets,
		runtime.AssetSpec{
			Name:      "tidb-cluster-build",
			PathKey:   "build_script",
			Content:   fmt.Sprintf("echo \"TiDB %s (%s) cluster compose stack started\"\n", service.Name, version),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:    "tidb-cluster-check",
			PathKey: "check_script",
			Content: fmt.Sprintf(
				"curl -fsS \"http://127.0.0.1:%d/status\"\n"+
					"echo\n"+
					"curl -fsS \"http://127.0.0.1:%d/status\"\n"+
					"echo\n"+
					"health_json=\"$(curl -fsS \"http://127.0.0.1:%d/pd/api/v1/health\")\"\n"+
					"echo \"$health_json\"\n"+
					"python3 - <<'PY' \"$health_json\"\nimport json,sys\nh=json.loads(sys.argv[1])\nif len(h)<3 or not all(x.get('health') for x in h):\n  raise SystemExit('PD health check failed')\nprint('pd_health_ok=true')\nPY\n"+
					"members_json=\"$(curl -fsS \"http://127.0.0.1:%d/pd/api/v1/members\")\"\n"+
					"echo \"$members_json\" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(\"members=\",len(d.get(\"members\",[])),\"leader=\",(d.get(\"leader\") or {}).get(\"name\"));'\n"+
					"stores=\"$(curl -fsS \"http://127.0.0.1:%d/pd/api/v1/stores\" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get(\"count\",0))')\"\n"+
					"[ \"${stores:-0}\" -ge 3 ]\n"+
					"for p in %d %d; do\n    if (exec 3<>/dev/tcp/127.0.0.1/$p) 2>/dev/null; then\n        echo \"tidb sql port $p is reachable\"\n        exec 3>&-\n    else\n        echo \"tidb sql port $p is not reachable\" >&2\n        exit 1\n    fi\ndone\n",
				tidb1StatusPort,
				tidb2StatusPort,
				pd1Port,
				pd1Port,
				pd1Port,
				tidb1Port,
				tidb2Port,
			),
			Mode:      0o755,
			MergeMode: runtime.AssetMergeScript,
		},
		runtime.AssetSpec{
			Name:      "tidb-cluster-readme",
			PathKey:   "readme_file",
			Content:   tidbClusterReadme(service, version, pdImage, tikvImage, tidbImage),
			Mode:      0o644,
			MergeMode: runtime.AssetMergeReadme,
		},
	)
	return context
}

func tidbClusterReadme(service model.BlueprintService, version, pdImage, tikvImage, tidbImage string) string {
	return fmt.Sprintf(
		"# TiDB %s Cluster\n\n- version: %s\n- pd image: %s\n- tikv image: %s\n- tidb image: %s\n- pd1 port: %d\n- pd2 port: %d\n- pd3 port: %d\n- tidb1 port: %d\n- tidb2 port: %d\n- tidb1 status port: %d\n- tidb2 status port: %d\n",
		service.Name,
		version,
		pdImage,
		tikvImage,
		tidbImage,
		service.Values[runtimecompose.ValuePD1Port].(int),
		service.Values[runtimecompose.ValuePD2Port].(int),
		service.Values[runtimecompose.ValuePD3Port].(int),
		service.Values[runtimecompose.ValueTiDB1Port].(int),
		service.Values[runtimecompose.ValueTiDB2Port].(int),
		service.Values[runtimecompose.ValueTiDB1StatusPort].(int),
		service.Values[runtimecompose.ValueTiDB2StatusPort].(int),
	)
}

func (*clusterSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template != clusterTemplate {
		return fmt.Errorf("tidb cluster validate: unexpected template %q", service.Template)
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("tidb cluster validate: unexpected middleware %q", service.Middleware)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("tidb cluster validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("tidb cluster validate version: %w", err)
	}

	for _, field := range []struct{ key, name string }{
		{runtimecompose.ValuePDImage, "pd_image"},
		{runtimecompose.ValueTiKVImage, "tikv_image"},
		{runtimecompose.ValueTiDBImage, "tidb_image"},
		{runtimecompose.ValuePD1ServiceName, "pd1_service_name"},
		{runtimecompose.ValuePD2ServiceName, "pd2_service_name"},
		{runtimecompose.ValuePD3ServiceName, "pd3_service_name"},
		{runtimecompose.ValueTiKV1ServiceName, "tikv1_service_name"},
		{runtimecompose.ValueTiKV2ServiceName, "tikv2_service_name"},
		{runtimecompose.ValueTiKV3ServiceName, "tikv3_service_name"},
		{runtimecompose.ValueTiDB1ServiceName, "tidb1_service_name"},
		{runtimecompose.ValueTiDB2ServiceName, "tidb2_service_name"},
		{runtimecompose.ValuePD1ContainerName, "pd1_container_name"},
		{runtimecompose.ValuePD2ContainerName, "pd2_container_name"},
		{runtimecompose.ValuePD3ContainerName, "pd3_container_name"},
		{runtimecompose.ValueTiKV1ContainerName, "tikv1_container_name"},
		{runtimecompose.ValueTiKV2ContainerName, "tikv2_container_name"},
		{runtimecompose.ValueTiKV3ContainerName, "tikv3_container_name"},
		{runtimecompose.ValueTiDB1ContainerName, "tidb1_container_name"},
		{runtimecompose.ValueTiDB2ContainerName, "tidb2_container_name"},
		{runtimecompose.ValuePD1DataDir, "pd1_data_dir"},
		{runtimecompose.ValuePD2DataDir, "pd2_data_dir"},
		{runtimecompose.ValuePD3DataDir, "pd3_data_dir"},
		{runtimecompose.ValueTiKV1DataDir, "tikv1_data_dir"},
		{runtimecompose.ValueTiKV2DataDir, "tikv2_data_dir"},
		{runtimecompose.ValueTiKV3DataDir, "tikv3_data_dir"},
	} {
		value, ok := service.Values[field.key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("tidb cluster validate %s: must be a non-empty string", field.name)
		}
	}

	portKeys := []struct{ key, name string }{
		{runtimecompose.ValuePD1Port, "pd1_port"},
		{runtimecompose.ValuePD2Port, "pd2_port"},
		{runtimecompose.ValuePD3Port, "pd3_port"},
		{runtimecompose.ValueTiDB1Port, "tidb1_port"},
		{runtimecompose.ValueTiDB2Port, "tidb2_port"},
		{runtimecompose.ValueTiDB1StatusPort, "tidb1_status_port"},
		{runtimecompose.ValueTiDB2StatusPort, "tidb2_status_port"},
	}
	seen := make(map[int]string, len(portKeys))
	for _, field := range portKeys {
		port, err := normalizePort(service.Values[field.key])
		if err != nil {
			return fmt.Errorf("tidb cluster validate %s: %w", field.name, err)
		}
		if port <= 0 {
			return fmt.Errorf("tidb cluster validate %s: must be greater than 0", field.name)
		}
		if previous, ok := seen[port]; ok {
			return fmt.Errorf("tidb cluster validate ports: %s and %s must be different", previous, field.name)
		}
		seen[port] = field.name
	}
	return nil
}

func (*clusterSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:            defaultVersion,
		runtimecompose.ValuePDImage:            "",
		runtimecompose.ValueTiKVImage:          "",
		runtimecompose.ValueTiDBImage:          "",
		runtimecompose.ValuePD1ServiceName:     "",
		runtimecompose.ValuePD2ServiceName:     "",
		runtimecompose.ValuePD3ServiceName:     "",
		runtimecompose.ValueTiKV1ServiceName:   "",
		runtimecompose.ValueTiKV2ServiceName:   "",
		runtimecompose.ValueTiKV3ServiceName:   "",
		runtimecompose.ValueTiDB1ServiceName:   "",
		runtimecompose.ValueTiDB2ServiceName:   "",
		runtimecompose.ValuePD1ContainerName:   "",
		runtimecompose.ValuePD2ContainerName:   "",
		runtimecompose.ValuePD3ContainerName:   "",
		runtimecompose.ValueTiKV1ContainerName: "",
		runtimecompose.ValueTiKV2ContainerName: "",
		runtimecompose.ValueTiKV3ContainerName: "",
		runtimecompose.ValueTiDB1ContainerName: "",
		runtimecompose.ValueTiDB2ContainerName: "",
		runtimecompose.ValuePD1DataDir:         "",
		runtimecompose.ValuePD2DataDir:         "",
		runtimecompose.ValuePD3DataDir:         "",
		runtimecompose.ValueTiKV1DataDir:       "",
		runtimecompose.ValueTiKV2DataDir:       "",
		runtimecompose.ValueTiKV3DataDir:       "",
		runtimecompose.ValuePD1Port:            defaultPD1Port,
		runtimecompose.ValuePD2Port:            defaultPD2Port,
		runtimecompose.ValuePD3Port:            defaultPD3Port,
		runtimecompose.ValueTiDB1Port:          defaultTiDB1Port,
		runtimecompose.ValueTiDB2Port:          defaultTiDB2Port,
		runtimecompose.ValueTiDB1StatusPort:    defaultTiDB1StatusPort,
		runtimecompose.ValueTiDB2StatusPort:    defaultTiDB2StatusPort,
	}
}
