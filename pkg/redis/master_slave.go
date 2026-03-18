package redis

import (
	"fmt"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
	"github.com/xuenqlve/zygarde/internal/tool"
)

const (
	masterSlaveTemplate = "master-slave"
	defaultMasterPort   = 6379
	defaultSlavePort    = 6380
)

// NewMasterSlaveSpec returns the default Redis master-slave middleware spec.
func NewMasterSlaveSpec() tpl.Middleware {
	return &masterSlaveSpec{}
}

type masterSlaveSpec struct {
	services []model.BlueprintService
}

func (*masterSlaveSpec) Middleware() string {
	return middlewareName
}

func (*masterSlaveSpec) Template() string {
	return masterSlaveTemplate
}

func (*masterSlaveSpec) IsDefault() bool {
	return false
}

func (s *masterSlaveSpec) Normalize(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	name := input.Name
	if name == "" {
		name = tpl.DefaultServiceName(s.Middleware(), index)
	}

	userSpecifiedMasterPort := hasValue(input.Values, runtimecompose.ValueMasterPort)
	userSpecifiedSlavePort := hasValue(input.Values, runtimecompose.ValueSlavePort)

	values := tpl.MergeValues(s.DefaultValues(), input.Values)
	version := defaultStringValue(values[runtimecompose.ValueVersion], defaultVersion)
	if err := validateVersion(version); err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize redis master-slave version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version

	masterServiceName := defaultStringValue(values[runtimecompose.ValueMasterServiceName], fmt.Sprintf("%s-master", name))
	slaveServiceName := defaultStringValue(values[runtimecompose.ValueSlaveServiceName], fmt.Sprintf("%s-slave", name))
	values[runtimecompose.ValueMasterServiceName] = masterServiceName
	values[runtimecompose.ValueSlaveServiceName] = slaveServiceName
	values[runtimecompose.ValueMasterContainerName] = defaultStringValue(values[runtimecompose.ValueMasterContainerName], masterServiceName)
	values[runtimecompose.ValueSlaveContainerName] = defaultStringValue(values[runtimecompose.ValueSlaveContainerName], slaveServiceName)
	values[runtimecompose.ValueMasterImage] = defaultStringValue(values[runtimecompose.ValueMasterImage], imageForVersion(version))
	values[runtimecompose.ValueSlaveImage] = defaultStringValue(values[runtimecompose.ValueSlaveImage], imageForVersion(version))
	values[runtimecompose.ValueMasterDataDir] = defaultStringValue(values[runtimecompose.ValueMasterDataDir], fmt.Sprintf("./data/%s", masterServiceName))
	values[runtimecompose.ValueSlaveDataDir] = defaultStringValue(values[runtimecompose.ValueSlaveDataDir], fmt.Sprintf("./data/%s", slaveServiceName))

	masterPort, err := allocateOrReservePort(values[runtimecompose.ValueMasterPort], userSpecifiedMasterPort, defaultMasterPort, "redis master-slave master_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueMasterPort] = masterPort

	slavePort, err := allocateOrReservePort(values[runtimecompose.ValueSlavePort], userSpecifiedSlavePort, defaultSlavePort, "redis master-slave slave_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueSlavePort] = slavePort

	return model.BlueprintService{
		Name:       name,
		Middleware: s.Middleware(),
		Template:   s.Template(),
		Values:     values,
	}, nil
}

func (s *masterSlaveSpec) Configure(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
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

func (s *masterSlaveSpec) BuildRuntimeContexts(runtimeType runtime.EnvironmentType) ([]runtime.EnvironmentContext, error) {
	services := append([]model.BlueprintService(nil), s.services...)
	s.services = nil

	contexts := make([]runtime.EnvironmentContext, 0, len(services)*2)
	for _, service := range services {
		if err := s.Validate(service); err != nil {
			return nil, err
		}

		version := service.Values[runtimecompose.ValueVersion].(string)
		masterServiceName := service.Values[runtimecompose.ValueMasterServiceName].(string)
		slaveServiceName := service.Values[runtimecompose.ValueSlaveServiceName].(string)
		masterContainerName := service.Values[runtimecompose.ValueMasterContainerName].(string)
		slaveContainerName := service.Values[runtimecompose.ValueSlaveContainerName].(string)
		masterImage := service.Values[runtimecompose.ValueMasterImage].(string)
		slaveImage := service.Values[runtimecompose.ValueSlaveImage].(string)
		masterDataDir := service.Values[runtimecompose.ValueMasterDataDir].(string)
		slaveDataDir := service.Values[runtimecompose.ValueSlaveDataDir].(string)
		masterPort, err := normalizePort(service.Values[runtimecompose.ValueMasterPort])
		if err != nil {
			return nil, fmt.Errorf("redis master-slave build runtime context master_port: %w", err)
		}
		slavePort, err := normalizePort(service.Values[runtimecompose.ValueSlavePort])
		if err != nil {
			return nil, fmt.Errorf("redis master-slave build runtime context slave_port: %w", err)
		}

		envKeyPrefix := serviceEnvKeyPrefix(service.Name)
		contexts = append(contexts,
			runtime.ComposeContext{
				EnvType:     runtimeType,
				ServiceName: masterServiceName,
				Middleware:  service.Middleware,
				Template:    service.Template,
				Service: runtime.ServiceSpec{
					Image:         masterImage,
					ContainerName: masterContainerName,
					Restart:       "unless-stopped",
					Ports:         []runtime.PortBinding{{HostPort: masterPort, ContainerPort: 6379, Protocol: "tcp"}},
					Volumes:       []runtime.VolumeMount{{Source: masterDataDir, Target: "/data"}},
					Command:       redisBaseCommand(),
					HealthCheck:   redisHealthCheck(),
				},
				Assets: []runtime.AssetSpec{
					{
						Name:    "redis-master-slave-env",
						PathKey: "env_file",
						Content: fmt.Sprintf(
							"%s_VERSION=%s\n%s_MASTER_IMAGE=%s\n%s_SLAVE_IMAGE=%s\n%s_MASTER_PORT=%d\n%s_SLAVE_PORT=%d\n%s_MASTER_CONTAINER_NAME=%s\n%s_SLAVE_CONTAINER_NAME=%s\n",
							envKeyPrefix, version,
							envKeyPrefix, masterImage,
							envKeyPrefix, slaveImage,
							envKeyPrefix, masterPort,
							envKeyPrefix, slavePort,
							envKeyPrefix, masterContainerName,
							envKeyPrefix, slaveContainerName,
						),
						Mode:      0o644,
						MergeMode: runtime.AssetMergeEnv,
					},
					{
						Name:      "redis-master-slave-build",
						PathKey:   "build_script",
						Content:   redisMasterSlaveBuildScript(service.Name, masterContainerName, slaveContainerName),
						Mode:      0o755,
						MergeMode: runtime.AssetMergeScript,
					},
					{
						Name:      "redis-master-slave-check",
						PathKey:   "check_script",
						Content:   redisMasterSlaveCheckScript(service.Name, masterContainerName, slaveContainerName),
						Mode:      0o755,
						MergeMode: runtime.AssetMergeScript,
					},
					{
						Name:      "redis-master-slave-readme",
						PathKey:   "readme_file",
						Content:   redisMasterSlaveReadme(service.Name, version, masterImage, slaveImage, masterPort, slavePort),
						Mode:      0o644,
						MergeMode: runtime.AssetMergeReadme,
					},
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			},
			runtime.ComposeContext{
				EnvType:     runtimeType,
				ServiceName: slaveServiceName,
				Middleware:  service.Middleware,
				Template:    service.Template,
				Service: runtime.ServiceSpec{
					Image:         slaveImage,
					ContainerName: slaveContainerName,
					Restart:       "unless-stopped",
					Ports:         []runtime.PortBinding{{HostPort: slavePort, ContainerPort: 6379, Protocol: "tcp"}},
					Volumes:       []runtime.VolumeMount{{Source: slaveDataDir, Target: "/data"}},
					Command:       append(redisBaseCommand(), "--replicaof", masterServiceName, "6379"),
					HealthCheck:   redisHealthCheck(),
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			},
		)
	}

	return contexts, nil
}

func (*masterSlaveSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template == "" {
		return tpl.ErrServiceTemplateRequired
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("redis master-slave validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != masterSlaveTemplate {
		return fmt.Errorf("redis master-slave validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("redis master-slave validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("redis master-slave validate version: %w", err)
	}

	fields := []struct {
		key  string
		name string
	}{
		{runtimecompose.ValueMasterServiceName, "master_service_name"},
		{runtimecompose.ValueSlaveServiceName, "slave_service_name"},
		{runtimecompose.ValueMasterContainerName, "master_container_name"},
		{runtimecompose.ValueSlaveContainerName, "slave_container_name"},
		{runtimecompose.ValueMasterImage, "master_image"},
		{runtimecompose.ValueSlaveImage, "slave_image"},
		{runtimecompose.ValueMasterDataDir, "master_data_dir"},
		{runtimecompose.ValueSlaveDataDir, "slave_data_dir"},
	}
	for _, field := range fields {
		value, ok := service.Values[field.key].(string)
		if !ok || value == "" {
			return fmt.Errorf("redis master-slave validate %s: must be a non-empty string", field.name)
		}
	}

	masterPort, err := normalizePort(service.Values[runtimecompose.ValueMasterPort])
	if err != nil {
		return fmt.Errorf("redis master-slave validate master_port: %w", err)
	}
	if masterPort <= 0 {
		return fmt.Errorf("redis master-slave validate master_port: must be greater than 0")
	}
	slavePort, err := normalizePort(service.Values[runtimecompose.ValueSlavePort])
	if err != nil {
		return fmt.Errorf("redis master-slave validate slave_port: %w", err)
	}
	if slavePort <= 0 {
		return fmt.Errorf("redis master-slave validate slave_port: must be greater than 0")
	}
	if masterPort == slavePort {
		return fmt.Errorf("redis master-slave validate ports: master_port and slave_port must be different")
	}

	return nil
}

func (*masterSlaveSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:       defaultVersion,
		runtimecompose.ValueMasterImage:   "",
		runtimecompose.ValueSlaveImage:    "",
		runtimecompose.ValueMasterDataDir: "",
		runtimecompose.ValueSlaveDataDir:  "",
		runtimecompose.ValueMasterPort:    defaultMasterPort,
		runtimecompose.ValueSlavePort:     defaultSlavePort,
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

func redisBaseCommand() []string {
	return []string{"redis-server", "--appendonly", "yes", "--save", "60 1000"}
}

func redisHealthCheck() *runtime.HealthCheck {
	return &runtime.HealthCheck{
		Test:        []string{"CMD", "redis-cli", "ping"},
		Interval:    "5s",
		Timeout:     "5s",
		Retries:     30,
		StartPeriod: "10s",
	}
}

func redisMasterSlaveBuildScript(name, masterContainerName, slaveContainerName string) string {
	return fmt.Sprintf(`wait_redis_healthy() {
    local name="$1"
    for _ in $(seq 1 30); do
        status="$("$CONTAINER_ENGINE" inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || true)"
        if [ "$status" = "healthy" ]; then
            return 0
        fi
        sleep 2
    done
    echo "container $name did not become healthy" >&2
    return 1
}

echo "[redis master-slave] waiting for %s master"
wait_redis_healthy %s

echo "[redis master-slave] waiting for %s slave"
wait_redis_healthy %s

echo "[redis master-slave] role summary for %s"
"$CONTAINER_ENGINE" exec %s redis-cli info replication | grep '^role:' || true
"$CONTAINER_ENGINE" exec %s redis-cli info replication | grep '^role:' || true
`, name, masterContainerName, name, slaveContainerName, name, masterContainerName, slaveContainerName)
}

func redisMasterSlaveCheckScript(name, masterContainerName, slaveContainerName string) string {
	return fmt.Sprintf(`echo "[redis master-slave] container status for %s"
"$CONTAINER_ENGINE" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /%s|%s/'

echo "[redis master-slave] connectivity for %s"
"$CONTAINER_ENGINE" exec %s redis-cli ping
"$CONTAINER_ENGINE" exec %s redis-cli ping

echo "[redis master-slave] master role for %s"
"$CONTAINER_ENGINE" exec %s redis-cli info replication | grep -E '^role:|connected_slaves:'

echo "[redis master-slave] slave role for %s"
"$CONTAINER_ENGINE" exec %s redis-cli info replication | grep -E '^role:|master_host:|master_link_status:'
`, name, masterContainerName, slaveContainerName, name, masterContainerName, slaveContainerName, name, masterContainerName, name, slaveContainerName)
}

func redisMasterSlaveReadme(name, version, masterImage, slaveImage string, masterPort, slavePort int) string {
	return fmt.Sprintf("# Redis %s\n\n- template: master-slave\n- version: %s\n- master image: %s\n- slave image: %s\n- master port: %d\n- slave port: %d\n", name, version, masterImage, slaveImage, masterPort, slavePort)
}
