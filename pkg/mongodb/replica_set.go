package mongodb

import (
	"fmt"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
	"github.com/xuenqlve/zygarde/internal/tool"
)

const (
	replicaSetTemplate = "replica-set"
	defaultRS1Port     = 27017
	defaultRS2Port     = 27018
	defaultRS3Port     = 27019
)

// NewReplicaSetSpec returns the default MongoDB replica-set middleware spec.
func NewReplicaSetSpec() tpl.Middleware {
	return &replicaSetSpec{}
}

type replicaSetSpec struct {
	services []model.BlueprintService
}

func (*replicaSetSpec) Middleware() string { return middlewareName }
func (*replicaSetSpec) Template() string   { return replicaSetTemplate }
func (*replicaSetSpec) IsDefault() bool    { return false }

func (s *replicaSetSpec) Normalize(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	name := input.Name
	if name == "" {
		name = tpl.DefaultServiceName(s.Middleware(), index)
	}

	values := tpl.MergeValues(s.DefaultValues(), input.Values)
	version := defaultStringValue(values[runtimecompose.ValueVersion], defaultVersion)
	if err := validateVersion(version); err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize mongodb replica-set version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))

	values[runtimecompose.ValueRS1ServiceName] = defaultStringValue(values[runtimecompose.ValueRS1ServiceName], fmt.Sprintf("%s-rs1", name))
	values[runtimecompose.ValueRS2ServiceName] = defaultStringValue(values[runtimecompose.ValueRS2ServiceName], fmt.Sprintf("%s-rs2", name))
	values[runtimecompose.ValueRS3ServiceName] = defaultStringValue(values[runtimecompose.ValueRS3ServiceName], fmt.Sprintf("%s-rs3", name))
	values[runtimecompose.ValueRS1ContainerName] = defaultStringValue(values[runtimecompose.ValueRS1ContainerName], values[runtimecompose.ValueRS1ServiceName].(string))
	values[runtimecompose.ValueRS2ContainerName] = defaultStringValue(values[runtimecompose.ValueRS2ContainerName], values[runtimecompose.ValueRS2ServiceName].(string))
	values[runtimecompose.ValueRS3ContainerName] = defaultStringValue(values[runtimecompose.ValueRS3ContainerName], values[runtimecompose.ValueRS3ServiceName].(string))
	values[runtimecompose.ValueRS1DataDir] = defaultStringValue(values[runtimecompose.ValueRS1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueRS1ServiceName].(string)))
	values[runtimecompose.ValueRS2DataDir] = defaultStringValue(values[runtimecompose.ValueRS2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueRS2ServiceName].(string)))
	values[runtimecompose.ValueRS3DataDir] = defaultStringValue(values[runtimecompose.ValueRS3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueRS3ServiceName].(string)))

	portSpecs := []struct {
		key      string
		fallback int
		name     string
	}{
		{runtimecompose.ValueRS1Port, defaultRS1Port, "mongodb replica-set rs1_port"},
		{runtimecompose.ValueRS2Port, defaultRS2Port, "mongodb replica-set rs2_port"},
		{runtimecompose.ValueRS3Port, defaultRS3Port, "mongodb replica-set rs3_port"},
	}
	for _, spec := range portSpecs {
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

func (s *replicaSetSpec) Configure(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
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

func (s *replicaSetSpec) BuildRuntimeContexts(runtimeType runtime.EnvironmentType) ([]runtime.EnvironmentContext, error) {
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
			serviceKey   string
			containerKey string
			dataDirKey   string
			portKey      string
		}{
			{runtimecompose.ValueRS1ServiceName, runtimecompose.ValueRS1ContainerName, runtimecompose.ValueRS1DataDir, runtimecompose.ValueRS1Port},
			{runtimecompose.ValueRS2ServiceName, runtimecompose.ValueRS2ContainerName, runtimecompose.ValueRS2DataDir, runtimecompose.ValueRS2Port},
			{runtimecompose.ValueRS3ServiceName, runtimecompose.ValueRS3ContainerName, runtimecompose.ValueRS3DataDir, runtimecompose.ValueRS3Port},
		}

		for index, node := range nodes {
			serviceName := service.Values[node.serviceKey].(string)
			containerName := service.Values[node.containerKey].(string)
			dataDir := service.Values[node.dataDirKey].(string)
			port, err := normalizePort(service.Values[node.portKey])
			if err != nil {
				return nil, fmt.Errorf("mongodb replica-set build runtime context port: %w", err)
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
						{HostPort: port, ContainerPort: 27017, Protocol: "tcp"},
					},
					Volumes: []runtime.VolumeMount{
						{Source: dataDir, Target: "/data/db"},
					},
					Command: []string{"mongod", "--replSet", "rs0", "--bind_ip_all", "--dbpath", "/data/db"},
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			}

			if index == 0 {
				rs1Port := service.Values[runtimecompose.ValueRS1Port].(int)
				rs2Port := service.Values[runtimecompose.ValueRS2Port].(int)
				rs3Port := service.Values[runtimecompose.ValueRS3Port].(int)
				rs1Container := service.Values[runtimecompose.ValueRS1ContainerName].(string)
				rs2Container := service.Values[runtimecompose.ValueRS2ContainerName].(string)
				rs3Container := service.Values[runtimecompose.ValueRS3ContainerName].(string)
				rs1Service := service.Values[runtimecompose.ValueRS1ServiceName].(string)
				rs2Service := service.Values[runtimecompose.ValueRS2ServiceName].(string)
				rs3Service := service.Values[runtimecompose.ValueRS3ServiceName].(string)

				context.Assets = []runtime.AssetSpec{
					{
						Name:    "mongodb-rs-env",
						PathKey: "env_file",
						Content: fmt.Sprintf(
							"%s_VERSION=%s\n%s_IMAGE=%s\n%s_RS1_PORT=%d\n%s_RS2_PORT=%d\n%s_RS3_PORT=%d\n%s_RS1_CONTAINER_NAME=%s\n%s_RS2_CONTAINER_NAME=%s\n%s_RS3_CONTAINER_NAME=%s\n",
							envKeyPrefix, version,
							envKeyPrefix, image,
							envKeyPrefix, rs1Port,
							envKeyPrefix, rs2Port,
							envKeyPrefix, rs3Port,
							envKeyPrefix, rs1Container,
							envKeyPrefix, rs2Container,
							envKeyPrefix, rs3Container,
						),
						Mode:      0o644,
						MergeMode: runtime.AssetMergeEnv,
					},
					{
						Name:      "mongodb-rs-build",
						PathKey:   "build_script",
						Content:   replicaSetBuildScript(service.Name, rs1Container, rs2Container, rs3Container, rs1Service, rs2Service, rs3Service),
						Mode:      0o755,
						MergeMode: runtime.AssetMergeScript,
					},
					{
						Name:      "mongodb-rs-check",
						PathKey:   "check_script",
						Content:   replicaSetCheckScript(service.Name, rs1Container, rs2Container, rs3Container),
						Mode:      0o755,
						MergeMode: runtime.AssetMergeScript,
					},
					{
						Name:      "mongodb-rs-readme",
						PathKey:   "readme_file",
						Content:   replicaSetReadme(service.Name, version, image, rs1Port, rs2Port, rs3Port),
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

func (*replicaSetSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template == "" {
		return tpl.ErrServiceTemplateRequired
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("mongodb replica-set validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != replicaSetTemplate {
		return fmt.Errorf("mongodb replica-set validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("mongodb replica-set validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("mongodb replica-set validate version: %w", err)
	}

	fields := []struct{ key, name string }{
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueRS1ServiceName, "rs1_service_name"},
		{runtimecompose.ValueRS2ServiceName, "rs2_service_name"},
		{runtimecompose.ValueRS3ServiceName, "rs3_service_name"},
		{runtimecompose.ValueRS1ContainerName, "rs1_container_name"},
		{runtimecompose.ValueRS2ContainerName, "rs2_container_name"},
		{runtimecompose.ValueRS3ContainerName, "rs3_container_name"},
		{runtimecompose.ValueRS1DataDir, "rs1_data_dir"},
		{runtimecompose.ValueRS2DataDir, "rs2_data_dir"},
		{runtimecompose.ValueRS3DataDir, "rs3_data_dir"},
	}
	for _, field := range fields {
		value, ok := service.Values[field.key].(string)
		if !ok || value == "" {
			return fmt.Errorf("mongodb replica-set validate %s: must be a non-empty string", field.name)
		}
	}

	portKeys := []struct{ key, name string }{
		{runtimecompose.ValueRS1Port, "rs1_port"},
		{runtimecompose.ValueRS2Port, "rs2_port"},
		{runtimecompose.ValueRS3Port, "rs3_port"},
	}
	seen := map[int]string{}
	for _, field := range portKeys {
		port, err := normalizePort(service.Values[field.key])
		if err != nil {
			return fmt.Errorf("mongodb replica-set validate %s: %w", field.name, err)
		}
		if port <= 0 {
			return fmt.Errorf("mongodb replica-set validate %s: must be greater than 0", field.name)
		}
		if prev, ok := seen[port]; ok {
			return fmt.Errorf("mongodb replica-set validate ports: %s conflicts with %s", field.name, prev)
		}
		seen[port] = field.name
	}

	return nil
}

func (*replicaSetSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:    defaultVersion,
		runtimecompose.ValueImage:      "",
		runtimecompose.ValueRS1DataDir: "",
		runtimecompose.ValueRS2DataDir: "",
		runtimecompose.ValueRS3DataDir: "",
		runtimecompose.ValueRS1Port:    defaultRS1Port,
		runtimecompose.ValueRS2Port:    defaultRS2Port,
		runtimecompose.ValueRS3Port:    defaultRS3Port,
	}
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

func replicaSetBuildScript(name, rs1Container, rs2Container, rs3Container, rs1Service, rs2Service, rs3Service string) string {
	return fmt.Sprintf(`echo "[mongodb replica-set] waiting for %s nodes"
for c in %s %s %s; do
  ok=0
  for _ in $(seq 1 30); do
    if "$CONTAINER_ENGINE" exec "$c" mongosh --quiet --eval 'db.adminCommand({ ping: 1 }).ok' >/dev/null 2>&1; then
      ok=1
      break
    fi
    sleep 2
  done
  [ "$ok" -eq 1 ] || { echo "$c not ready" >&2; exit 1; }
done

echo "[mongodb replica-set] initiating rs0 for %s"
"$CONTAINER_ENGINE" exec %s mongosh --quiet --eval '
try {
  rs.initiate({_id:"rs0", members:[
    {_id:0, host:"%s:27017"},
    {_id:1, host:"%s:27017"},
    {_id:2, host:"%s:27017"}
  ]})
} catch(e) {
  if (!e.message.includes("already initialized")) throw e;
}
'

echo "[mongodb replica-set] waiting for PRIMARY/SECONDARY for %s"
ok=0
for _ in $(seq 1 60); do
  line="$("$CONTAINER_ENGINE" exec %s mongosh --quiet --eval '
try {
  var s=rs.status();
  var p=s.members.filter(m=>m.stateStr=="PRIMARY").length;
  var sec=s.members.filter(m=>m.stateStr=="SECONDARY").length;
  print(p+","+sec);
} catch(e) { print("0,0"); }
')"
  p="${line%%,*}"
  sec="${line##*,}"
  [ "$p" = "1" ] && [ "$sec" -ge 2 ] && ok=1 && break
  sleep 2
done
if [ "$ok" -ne 1 ]; then
  "$CONTAINER_ENGINE" exec %s mongosh --quiet --eval 'try{print(JSON.stringify(rs.status().members.map(m=>({name:m.name,stateStr:m.stateStr,health:m.health}))))}catch(e){print(e.message)}' >&2 || true
  exit 1
fi
`, name, rs1Container, rs2Container, rs3Container, name, rs1Container, rs1Service, rs2Service, rs3Service, name, rs1Container, rs1Container)
}

func replicaSetCheckScript(name, rs1Container, rs2Container, rs3Container string) string {
	return fmt.Sprintf(`echo "[mongodb replica-set] container status for %s"
"$CONTAINER_ENGINE" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /%s|%s|%s/'

echo "[mongodb replica-set] connectivity for %s"
"$CONTAINER_ENGINE" exec %s mongosh --quiet --eval 'db.adminCommand({ ping: 1 })'

echo "[mongodb replica-set] replica-set status for %s"
"$CONTAINER_ENGINE" exec %s mongosh --quiet --eval 'JSON.stringify(rs.status().members.map(m=>({name:m.name,stateStr:m.stateStr})))'

echo "[mongodb replica-set] primary check for %s"
PRIMARY="$("$CONTAINER_ENGINE" exec %s mongosh --quiet --eval 'rs.status().members.filter(m=>m.stateStr=="PRIMARY").length')"
if [ "$PRIMARY" -lt 1 ]; then
  echo "No PRIMARY found" >&2
  exit 1
fi
`, name, rs1Container, rs2Container, rs3Container, name, rs1Container, name, rs1Container, name, rs1Container)
}

func replicaSetReadme(name, version, image string, rs1Port, rs2Port, rs3Port int) string {
	return fmt.Sprintf("# MongoDB %s\n\n- template: replica-set\n- version: %s\n- image: %s\n- rs ports: %d, %d, %d\n", name, version, image, rs1Port, rs2Port, rs3Port)
}
