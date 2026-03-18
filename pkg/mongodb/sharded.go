package mongodb

import (
	"fmt"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

const (
	shardedTemplate   = "sharded"
	defaultMongosPort = 27017
	configServerPort  = 27019
	shardServerPort   = 27018
)

// NewShardedSpec returns the default MongoDB sharded middleware spec.
func NewShardedSpec() tpl.Middleware {
	return &shardedSpec{}
}

type shardedSpec struct {
	services []model.BlueprintService
}

func (*shardedSpec) Middleware() string { return middlewareName }
func (*shardedSpec) Template() string   { return shardedTemplate }
func (*shardedSpec) IsDefault() bool    { return false }

func (s *shardedSpec) Normalize(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	name := input.Name
	if name == "" {
		name = tpl.DefaultServiceName(s.Middleware(), index)
	}

	values := tpl.MergeValues(s.DefaultValues(), input.Values)
	version := defaultStringValue(values[runtimecompose.ValueVersion], defaultVersion)
	if err := validateVersion(version); err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize mongodb sharded version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version
	values[runtimecompose.ValueImage] = defaultStringValue(values[runtimecompose.ValueImage], imageForVersion(version))

	values[runtimecompose.ValueCfg1ServiceName] = defaultStringValue(values[runtimecompose.ValueCfg1ServiceName], fmt.Sprintf("%s-cfg1", name))
	values[runtimecompose.ValueCfg2ServiceName] = defaultStringValue(values[runtimecompose.ValueCfg2ServiceName], fmt.Sprintf("%s-cfg2", name))
	values[runtimecompose.ValueCfg3ServiceName] = defaultStringValue(values[runtimecompose.ValueCfg3ServiceName], fmt.Sprintf("%s-cfg3", name))
	values[runtimecompose.ValueShard1ServiceName] = defaultStringValue(values[runtimecompose.ValueShard1ServiceName], fmt.Sprintf("%s-shard1", name))
	values[runtimecompose.ValueShard2ServiceName] = defaultStringValue(values[runtimecompose.ValueShard2ServiceName], fmt.Sprintf("%s-shard2", name))
	values[runtimecompose.ValueMongosServiceName] = defaultStringValue(values[runtimecompose.ValueMongosServiceName], fmt.Sprintf("%s-mongos", name))

	values[runtimecompose.ValueCfg1ContainerName] = defaultStringValue(values[runtimecompose.ValueCfg1ContainerName], values[runtimecompose.ValueCfg1ServiceName].(string))
	values[runtimecompose.ValueCfg2ContainerName] = defaultStringValue(values[runtimecompose.ValueCfg2ContainerName], values[runtimecompose.ValueCfg2ServiceName].(string))
	values[runtimecompose.ValueCfg3ContainerName] = defaultStringValue(values[runtimecompose.ValueCfg3ContainerName], values[runtimecompose.ValueCfg3ServiceName].(string))
	values[runtimecompose.ValueShard1ContainerName] = defaultStringValue(values[runtimecompose.ValueShard1ContainerName], values[runtimecompose.ValueShard1ServiceName].(string))
	values[runtimecompose.ValueShard2ContainerName] = defaultStringValue(values[runtimecompose.ValueShard2ContainerName], values[runtimecompose.ValueShard2ServiceName].(string))
	values[runtimecompose.ValueMongosContainerName] = defaultStringValue(values[runtimecompose.ValueMongosContainerName], values[runtimecompose.ValueMongosServiceName].(string))

	values[runtimecompose.ValueCfg1DataDir] = defaultStringValue(values[runtimecompose.ValueCfg1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueCfg1ServiceName].(string)))
	values[runtimecompose.ValueCfg2DataDir] = defaultStringValue(values[runtimecompose.ValueCfg2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueCfg2ServiceName].(string)))
	values[runtimecompose.ValueCfg3DataDir] = defaultStringValue(values[runtimecompose.ValueCfg3DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueCfg3ServiceName].(string)))
	values[runtimecompose.ValueShard1DataDir] = defaultStringValue(values[runtimecompose.ValueShard1DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueShard1ServiceName].(string)))
	values[runtimecompose.ValueShard2DataDir] = defaultStringValue(values[runtimecompose.ValueShard2DataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueShard2ServiceName].(string)))

	mongosPort, err := allocateOrReservePort(values[runtimecompose.ValueMongosPort], hasValue(input.Values, runtimecompose.ValueMongosPort), defaultMongosPort, "mongodb sharded mongos_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueMongosPort] = mongosPort

	return model.BlueprintService{
		Name:       name,
		Middleware: s.Middleware(),
		Template:   s.Template(),
		Values:     values,
	}, nil
}

func (s *shardedSpec) Configure(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
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

func (s *shardedSpec) BuildRuntimeContexts(runtimeType runtime.EnvironmentType) ([]runtime.EnvironmentContext, error) {
	services := append([]model.BlueprintService(nil), s.services...)
	s.services = nil

	contexts := make([]runtime.EnvironmentContext, 0, len(services)*6)
	for _, service := range services {
		if err := s.Validate(service); err != nil {
			return nil, err
		}

		version := service.Values[runtimecompose.ValueVersion].(string)
		image := service.Values[runtimecompose.ValueImage].(string)
		envKeyPrefix := serviceEnvKeyPrefix(service.Name)

		contexts = append(contexts,
			mongoConfigServerContext(runtimeType, service, image, runtimecompose.ValueCfg1ServiceName, runtimecompose.ValueCfg1ContainerName, runtimecompose.ValueCfg1DataDir),
			mongoConfigServerContext(runtimeType, service, image, runtimecompose.ValueCfg2ServiceName, runtimecompose.ValueCfg2ContainerName, runtimecompose.ValueCfg2DataDir),
			mongoConfigServerContext(runtimeType, service, image, runtimecompose.ValueCfg3ServiceName, runtimecompose.ValueCfg3ContainerName, runtimecompose.ValueCfg3DataDir),
			mongoShardServerContext(runtimeType, service, image, runtimecompose.ValueShard1ServiceName, runtimecompose.ValueShard1ContainerName, runtimecompose.ValueShard1DataDir),
			mongoShardServerContext(runtimeType, service, image, runtimecompose.ValueShard2ServiceName, runtimecompose.ValueShard2ContainerName, runtimecompose.ValueShard2DataDir),
		)

		mongosPort := service.Values[runtimecompose.ValueMongosPort].(int)
		cfg1Container := service.Values[runtimecompose.ValueCfg1ContainerName].(string)
		cfg2Container := service.Values[runtimecompose.ValueCfg2ContainerName].(string)
		cfg3Container := service.Values[runtimecompose.ValueCfg3ContainerName].(string)
		shard1Container := service.Values[runtimecompose.ValueShard1ContainerName].(string)
		shard2Container := service.Values[runtimecompose.ValueShard2ContainerName].(string)
		cfg1Service := service.Values[runtimecompose.ValueCfg1ServiceName].(string)
		cfg2Service := service.Values[runtimecompose.ValueCfg2ServiceName].(string)
		cfg3Service := service.Values[runtimecompose.ValueCfg3ServiceName].(string)
		shard1Service := service.Values[runtimecompose.ValueShard1ServiceName].(string)
		shard2Service := service.Values[runtimecompose.ValueShard2ServiceName].(string)
		mongosService := service.Values[runtimecompose.ValueMongosServiceName].(string)
		mongosContainer := service.Values[runtimecompose.ValueMongosContainerName].(string)

		contexts = append(contexts, runtime.ComposeContext{
			EnvType:     runtimeType,
			ServiceName: mongosService,
			Middleware:  service.Middleware,
			Template:    service.Template,
			Service: runtime.ServiceSpec{
				Image:         image,
				ContainerName: mongosContainer,
				Restart:       "unless-stopped",
				Ports: []runtime.PortBinding{
					{HostPort: mongosPort, ContainerPort: 27017, Protocol: "tcp"},
				},
				Command: []string{
					"mongos",
					"--configdb", fmt.Sprintf("cfgRS/%s:%d,%s:%d,%s:%d", cfg1Service, configServerPort, cfg2Service, configServerPort, cfg3Service, configServerPort),
					"--bind_ip_all",
					"--port", "27017",
				},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "mongodb-sharded-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"%s_VERSION=%s\n%s_IMAGE=%s\n%s_MONGOS_PORT=%d\n%s_CFG1_CONTAINER_NAME=%s\n%s_CFG2_CONTAINER_NAME=%s\n%s_CFG3_CONTAINER_NAME=%s\n%s_SHARD1_CONTAINER_NAME=%s\n%s_SHARD2_CONTAINER_NAME=%s\n",
						envKeyPrefix, version,
						envKeyPrefix, image,
						envKeyPrefix, mongosPort,
						envKeyPrefix, cfg1Container,
						envKeyPrefix, cfg2Container,
						envKeyPrefix, cfg3Container,
						envKeyPrefix, shard1Container,
						envKeyPrefix, shard2Container,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "mongodb-sharded-build",
					PathKey:   "build_script",
					Content:   shardedBuildScript(service.Name, cfg1Container, cfg2Container, cfg3Container, shard1Container, shard2Container, mongosContainer, cfg1Service, cfg2Service, cfg3Service, shard1Service, shard2Service),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "mongodb-sharded-check",
					PathKey:   "check_script",
					Content:   shardedCheckScript(service.Name, cfg1Container, cfg2Container, cfg3Container, shard1Container, shard2Container, mongosContainer),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "mongodb-sharded-readme",
					PathKey:   "readme_file",
					Content:   shardedReadme(service.Name, version, image, mongosPort),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeReadme,
				},
			},
			Metadata: tpl.MergeValues(nil, service.Values),
		})
	}

	return contexts, nil
}

func mongoConfigServerContext(runtimeType runtime.EnvironmentType, service model.BlueprintService, image, serviceKey, containerKey, dataDirKey string) runtime.ComposeContext {
	return runtime.ComposeContext{
		EnvType:     runtimeType,
		ServiceName: service.Values[serviceKey].(string),
		Middleware:  service.Middleware,
		Template:    service.Template,
		Service: runtime.ServiceSpec{
			Image:         image,
			ContainerName: service.Values[containerKey].(string),
			Restart:       "unless-stopped",
			Volumes:       []runtime.VolumeMount{{Source: service.Values[dataDirKey].(string), Target: "/data/db"}},
			Command:       []string{"mongod", "--configsvr", "--replSet", "cfgRS", "--bind_ip_all", "--port", fmt.Sprintf("%d", configServerPort)},
		},
		Metadata: tpl.MergeValues(nil, service.Values),
	}
}

func mongoShardServerContext(runtimeType runtime.EnvironmentType, service model.BlueprintService, image, serviceKey, containerKey, dataDirKey string) runtime.ComposeContext {
	return runtime.ComposeContext{
		EnvType:     runtimeType,
		ServiceName: service.Values[serviceKey].(string),
		Middleware:  service.Middleware,
		Template:    service.Template,
		Service: runtime.ServiceSpec{
			Image:         image,
			ContainerName: service.Values[containerKey].(string),
			Restart:       "unless-stopped",
			Volumes:       []runtime.VolumeMount{{Source: service.Values[dataDirKey].(string), Target: "/data/db"}},
			Command:       []string{"mongod", "--shardsvr", "--replSet", "shardRS", "--bind_ip_all", "--port", fmt.Sprintf("%d", shardServerPort)},
		},
		Metadata: tpl.MergeValues(nil, service.Values),
	}
}

func (*shardedSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template == "" {
		return tpl.ErrServiceTemplateRequired
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("mongodb sharded validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != shardedTemplate {
		return fmt.Errorf("mongodb sharded validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("mongodb sharded validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("mongodb sharded validate version: %w", err)
	}

	stringFields := []struct{ key, name string }{
		{runtimecompose.ValueImage, "image"},
		{runtimecompose.ValueCfg1ServiceName, "cfg1_service_name"},
		{runtimecompose.ValueCfg2ServiceName, "cfg2_service_name"},
		{runtimecompose.ValueCfg3ServiceName, "cfg3_service_name"},
		{runtimecompose.ValueCfg1ContainerName, "cfg1_container_name"},
		{runtimecompose.ValueCfg2ContainerName, "cfg2_container_name"},
		{runtimecompose.ValueCfg3ContainerName, "cfg3_container_name"},
		{runtimecompose.ValueCfg1DataDir, "cfg1_data_dir"},
		{runtimecompose.ValueCfg2DataDir, "cfg2_data_dir"},
		{runtimecompose.ValueCfg3DataDir, "cfg3_data_dir"},
		{runtimecompose.ValueShard1ServiceName, "shard1_service_name"},
		{runtimecompose.ValueShard2ServiceName, "shard2_service_name"},
		{runtimecompose.ValueShard1ContainerName, "shard1_container_name"},
		{runtimecompose.ValueShard2ContainerName, "shard2_container_name"},
		{runtimecompose.ValueShard1DataDir, "shard1_data_dir"},
		{runtimecompose.ValueShard2DataDir, "shard2_data_dir"},
		{runtimecompose.ValueMongosServiceName, "mongos_service_name"},
		{runtimecompose.ValueMongosContainerName, "mongos_container_name"},
	}
	for _, field := range stringFields {
		value, ok := service.Values[field.key].(string)
		if !ok || value == "" {
			return fmt.Errorf("mongodb sharded validate %s: must be a non-empty string", field.name)
		}
	}

	mongosPort, err := normalizePort(service.Values[runtimecompose.ValueMongosPort])
	if err != nil {
		return fmt.Errorf("mongodb sharded validate mongos_port: %w", err)
	}
	if mongosPort <= 0 {
		return fmt.Errorf("mongodb sharded validate mongos_port: must be greater than 0")
	}
	return nil
}

func (*shardedSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:       defaultVersion,
		runtimecompose.ValueImage:         "",
		runtimecompose.ValueCfg1DataDir:   "",
		runtimecompose.ValueCfg2DataDir:   "",
		runtimecompose.ValueCfg3DataDir:   "",
		runtimecompose.ValueShard1DataDir: "",
		runtimecompose.ValueShard2DataDir: "",
		runtimecompose.ValueMongosPort:    defaultMongosPort,
	}
}

func shardedBuildScript(name, cfg1Container, cfg2Container, cfg3Container, shard1Container, shard2Container, mongosContainer, cfg1Service, cfg2Service, cfg3Service, shard1Service, shard2Service string) string {
	return fmt.Sprintf(`wait_ping() {
  local c="$1"; local port="$2"; local retries="${3:-40}"
  local ok=0
  for _ in $(seq 1 "$retries"); do
    if "$CONTAINER_ENGINE" exec "$c" mongosh --quiet --port "$port" --eval 'db.adminCommand({ ping: 1 }).ok' >/dev/null 2>&1; then
      ok=1; break
    fi
    sleep 2
  done
  [ "$ok" -eq 1 ]
}

wait_rs_primary() {
  local c="$1"; local port="$2"; local retries="${3:-60}"
  for _ in $(seq 1 "$retries"); do
    state="$("$CONTAINER_ENGINE" exec "$c" mongosh --quiet --port "$port" --eval 'try{rs.status().myState}catch(e){0}' 2>/dev/null | tail -n 1 || true)"
    if [ "$state" = "1" ]; then
      return 0
    fi
    sleep 2
  done
  return 1
}

echo "[mongodb sharded] waiting for %s topology"
wait_ping %s %d 50 || exit 1
wait_ping %s %d 50 || exit 1
wait_ping %s %d 50 || exit 1
wait_ping %s %d 50 || exit 1
wait_ping %s %d 50 || exit 1

echo "[mongodb sharded] initiating cfgRS for %s"
"$CONTAINER_ENGINE" exec %s mongosh --quiet --port %d --eval '
try {
  rs.initiate({_id:"cfgRS", configsvr:true, members:[
    {_id:0, host:"%s:%d"},
    {_id:1, host:"%s:%d"},
    {_id:2, host:"%s:%d"}
  ]})
} catch(e) { if (!e.message.includes("already initialized")) throw e; }
'
wait_rs_primary %s %d 70 || exit 1

echo "[mongodb sharded] initiating shardRS for %s"
"$CONTAINER_ENGINE" exec %s mongosh --quiet --port %d --eval '
try {
  rs.initiate({_id:"shardRS", members:[
    {_id:0, host:"%s:%d"},
    {_id:1, host:"%s:%d"}
  ]})
} catch(e) { if (!e.message.includes("already initialized")) throw e; }
'
wait_rs_primary %s %d 70 || exit 1

wait_ping %s %d 60 || exit 1

echo "[mongodb sharded] add shard for %s"
"$CONTAINER_ENGINE" exec %s mongosh --quiet --port %d --eval '
try {
  sh.addShard("shardRS/%s:%d,%s:%d")
} catch(e) {
  if (!(e.message.includes("already") || e.message.includes("exists"))) throw e;
}
sh.status()
'
`, name,
		cfg1Container, configServerPort,
		cfg2Container, configServerPort,
		cfg3Container, configServerPort,
		shard1Container, shardServerPort,
		shard2Container, shardServerPort,
		name, cfg1Container, configServerPort,
		cfg1Service, configServerPort,
		cfg2Service, configServerPort,
		cfg3Service, configServerPort,
		cfg1Container, configServerPort,
		name, shard1Container, shardServerPort,
		shard1Service, shardServerPort,
		shard2Service, shardServerPort,
		shard1Container, shardServerPort,
		mongosContainer, 27017,
		name, mongosContainer, 27017,
		shard1Service, shardServerPort,
		shard2Service, shardServerPort)
}

func shardedCheckScript(name, cfg1Container, cfg2Container, cfg3Container, shard1Container, shard2Container, mongosContainer string) string {
	return fmt.Sprintf(`echo "[mongodb sharded] container status for %s"
"$CONTAINER_ENGINE" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /%s|%s|%s|%s|%s|%s/'

echo "[mongodb sharded] mongos ping for %s"
"$CONTAINER_ENGINE" exec %s mongosh --quiet --port 27017 --eval 'db.adminCommand({ ping: 1 })'

echo "[mongodb sharded] listShards for %s"
SHARDS="$("$CONTAINER_ENGINE" exec %s mongosh --quiet --port 27017 --eval 'JSON.stringify(db.adminCommand({ listShards: 1 }))')"
echo "$SHARDS"
echo "$SHARDS" | grep -q '"ok":1' || { echo "listShards not ok" >&2; exit 1; }

echo "[mongodb sharded] shard count for %s"
COUNT="$("$CONTAINER_ENGINE" exec %s mongosh --quiet --port 27017 --eval 'db.adminCommand({ listShards: 1 }).shards.length')"
if [ "$COUNT" -lt 1 ]; then
  echo "No shard found" >&2
  exit 1
fi
`, name, cfg1Container, cfg2Container, cfg3Container, shard1Container, shard2Container, mongosContainer, name, mongosContainer, name, mongosContainer, name, mongosContainer)
}

func shardedReadme(name, version, image string, mongosPort int) string {
	return fmt.Sprintf("# MongoDB %s\n\n- template: sharded\n- version: %s\n- image: %s\n- mongos port: %d\n", name, version, image, mongosPort)
}
