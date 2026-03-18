package mysql

import (
	"fmt"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
	"github.com/xuenqlve/zygarde/internal/tool"
)

const (
	masterSlaveTemplate        = "master-slave"
	defaultMasterPort          = 3306
	defaultSlavePort           = 3307
	defaultReplicationUser     = "repl"
	defaultReplicationPassword = "repl123"
	defaultMasterServerID      = 1
	defaultSlaveServerID       = 2
)

// NewMasterSlaveSpec returns the default MySQL master-slave middleware spec.
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
		return model.BlueprintService{}, fmt.Errorf("normalize mysql master-slave version: %w", err)
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

	masterPort, err := allocateOrReservePort(
		values[runtimecompose.ValueMasterPort],
		userSpecifiedMasterPort,
		defaultMasterPort,
		"mysql master-slave master_port",
	)
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueMasterPort] = masterPort

	slavePort, err := allocateOrReservePort(
		values[runtimecompose.ValueSlavePort],
		userSpecifiedSlavePort,
		defaultSlavePort,
		"mysql master-slave slave_port",
	)
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueSlavePort] = slavePort

	rootPassword := defaultStringValue(values[runtimecompose.ValueRootPassword], "")
	if rootPassword == "" {
		return model.BlueprintService{}, fmt.Errorf("normalize mysql master-slave root_password: must be a non-empty string")
	}
	values[runtimecompose.ValueRootPassword] = rootPassword

	replicationUser := defaultStringValue(values[runtimecompose.ValueReplicationUser], defaultReplicationUser)
	if replicationUser == "" {
		return model.BlueprintService{}, fmt.Errorf("normalize mysql master-slave replication_user: must be a non-empty string")
	}
	values[runtimecompose.ValueReplicationUser] = replicationUser

	replicationPassword := defaultStringValue(values[runtimecompose.ValueReplicationPassword], defaultReplicationPassword)
	if replicationPassword == "" {
		return model.BlueprintService{}, fmt.Errorf("normalize mysql master-slave replication_password: must be a non-empty string")
	}
	values[runtimecompose.ValueReplicationPassword] = replicationPassword

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
		rootPassword := service.Values[runtimecompose.ValueRootPassword].(string)
		replicationUser := service.Values[runtimecompose.ValueReplicationUser].(string)
		replicationPassword := service.Values[runtimecompose.ValueReplicationPassword].(string)
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
			return nil, fmt.Errorf("mysql master-slave build runtime context master_port: %w", err)
		}
		slavePort, err := normalizePort(service.Values[runtimecompose.ValueSlavePort])
		if err != nil {
			return nil, fmt.Errorf("mysql master-slave build runtime context slave_port: %w", err)
		}

		envKeyPrefix := serviceEnvKeyPrefix(service.Name)
		masterInitFile := fmt.Sprintf("%s-master-init.sql", service.Name)
		slaveInitFile := fmt.Sprintf("%s-slave-init.sql", service.Name)

		contexts = append(contexts, runtime.ComposeContext{
			EnvType:     runtimeType,
			ServiceName: masterServiceName,
			Middleware:  service.Middleware,
			Template:    service.Template,
			Service: runtime.ServiceSpec{
				Image:         masterImage,
				Platform:      platformForVersion(version),
				ContainerName: masterContainerName,
				Restart:       "unless-stopped",
				Environment: map[string]string{
					"MYSQL_ROOT_PASSWORD": rootPassword,
					"MYSQL_ROOT_HOST":     "%",
				},
				Ports: []runtime.PortBinding{
					{
						HostPort:      masterPort,
						ContainerPort: 3306,
						Protocol:      "tcp",
					},
				},
				Volumes: []runtime.VolumeMount{
					{
						Source: masterDataDir,
						Target: "/var/lib/mysql",
					},
					{
						Source:   "./" + masterInitFile,
						Target:   "/docker-entrypoint-initdb.d/01-master-init.sql",
						ReadOnly: true,
					},
				},
				Command:     masterCommandForVersion(version),
				HealthCheck: mysqlHealthCheck(rootPassword),
			},
			Assets: []runtime.AssetSpec{
				{
					Name:    "mysql-master-slave-env",
					PathKey: "env_file",
					Content: fmt.Sprintf(
						"%s_VERSION=%s\n%s_MASTER_IMAGE=%s\n%s_SLAVE_IMAGE=%s\n%s_MASTER_PORT=%d\n%s_SLAVE_PORT=%d\n%s_ROOT_PASSWORD=%s\n%s_REPLICATION_USER=%s\n%s_REPLICATION_PASSWORD=%s\n%s_MASTER_CONTAINER_NAME=%s\n%s_SLAVE_CONTAINER_NAME=%s\n",
						envKeyPrefix,
						version,
						envKeyPrefix,
						masterImage,
						envKeyPrefix,
						slaveImage,
						envKeyPrefix,
						masterPort,
						envKeyPrefix,
						slavePort,
						envKeyPrefix,
						rootPassword,
						envKeyPrefix,
						replicationUser,
						envKeyPrefix,
						replicationPassword,
						envKeyPrefix,
						masterContainerName,
						envKeyPrefix,
						slaveContainerName,
					),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeEnv,
				},
				{
					Name:      "mysql-master-slave-build",
					PathKey:   "build_script",
					Content:   masterSlaveBuildScript(service.Name, envKeyPrefix, masterContainerName, slaveContainerName, slaveInitFile, version),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "mysql-master-slave-check",
					PathKey:   "check_script",
					Content:   masterSlaveCheckScript(service.Name, envKeyPrefix, masterContainerName, slaveContainerName),
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "mysql-master-slave-readme",
					PathKey:   "readme_file",
					Content:   masterSlaveReadme(service.Name, version, masterImage, slaveImage, masterPort, slavePort),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeReadme,
				},
				{
					Name:      "mysql-master-init",
					FileName:  masterInitFile,
					Content:   masterInitSQL(replicationUser, replicationPassword),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeUnique,
				},
				{
					Name:      "mysql-slave-init",
					FileName:  slaveInitFile,
					Content:   slaveInitSQL(version, masterServiceName, replicationUser, replicationPassword),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeUnique,
				},
			},
			Metadata: tpl.MergeValues(nil, service.Values),
		})

		contexts = append(contexts, runtime.ComposeContext{
			EnvType:     runtimeType,
			ServiceName: slaveServiceName,
			Middleware:  service.Middleware,
			Template:    service.Template,
			Service: runtime.ServiceSpec{
				Image:         slaveImage,
				Platform:      platformForVersion(version),
				ContainerName: slaveContainerName,
				Restart:       "unless-stopped",
				Environment: map[string]string{
					"MYSQL_ROOT_PASSWORD": rootPassword,
					"MYSQL_ROOT_HOST":     "%",
				},
				Ports: []runtime.PortBinding{
					{
						HostPort:      slavePort,
						ContainerPort: 3306,
						Protocol:      "tcp",
					},
				},
				Volumes: []runtime.VolumeMount{
					{
						Source: slaveDataDir,
						Target: "/var/lib/mysql",
					},
				},
				Command:     slaveCommandForVersion(version),
				HealthCheck: mysqlHealthCheck(rootPassword),
			},
			Metadata: tpl.MergeValues(nil, service.Values),
		})
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
		return fmt.Errorf("mysql master-slave validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != masterSlaveTemplate {
		return fmt.Errorf("mysql master-slave validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("mysql master-slave validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("mysql master-slave validate version: %w", err)
	}

	rootPassword, ok := service.Values[runtimecompose.ValueRootPassword].(string)
	if !ok || rootPassword == "" {
		return fmt.Errorf("mysql master-slave validate root_password: must be a non-empty string")
	}

	replicationUser, ok := service.Values[runtimecompose.ValueReplicationUser].(string)
	if !ok || replicationUser == "" {
		return fmt.Errorf("mysql master-slave validate replication_user: must be a non-empty string")
	}

	replicationPassword, ok := service.Values[runtimecompose.ValueReplicationPassword].(string)
	if !ok || replicationPassword == "" {
		return fmt.Errorf("mysql master-slave validate replication_password: must be a non-empty string")
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
			return fmt.Errorf("mysql master-slave validate %s: must be a non-empty string", field.name)
		}
	}

	masterPort, err := normalizePort(service.Values[runtimecompose.ValueMasterPort])
	if err != nil {
		return fmt.Errorf("mysql master-slave validate master_port: %w", err)
	}
	if masterPort <= 0 {
		return fmt.Errorf("mysql master-slave validate master_port: must be greater than 0")
	}

	slavePort, err := normalizePort(service.Values[runtimecompose.ValueSlavePort])
	if err != nil {
		return fmt.Errorf("mysql master-slave validate slave_port: %w", err)
	}
	if slavePort <= 0 {
		return fmt.Errorf("mysql master-slave validate slave_port: must be greater than 0")
	}
	if masterPort == slavePort {
		return fmt.Errorf("mysql master-slave validate ports: master_port and slave_port must be different")
	}

	return nil
}

func (*masterSlaveSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:             defaultVersion,
		runtimecompose.ValueRootPassword:        "root",
		runtimecompose.ValueMasterImage:         "",
		runtimecompose.ValueSlaveImage:          "",
		runtimecompose.ValueMasterDataDir:       "",
		runtimecompose.ValueSlaveDataDir:        "",
		runtimecompose.ValueMasterPort:          defaultMasterPort,
		runtimecompose.ValueSlavePort:           defaultSlavePort,
		runtimecompose.ValueReplicationUser:     defaultReplicationUser,
		runtimecompose.ValueReplicationPassword: defaultReplicationPassword,
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

func masterCommandForVersion(version string) []string {
	command := []string{
		fmt.Sprintf("--server-id=%d", defaultMasterServerID),
		"--log-bin=mysql-bin",
		"--binlog-format=ROW",
		"--gtid-mode=ON",
		"--enforce-gtid-consistency=ON",
		"--skip-name-resolve=1",
		"--default-authentication-plugin=mysql_native_password",
	}
	if version == "v8.0" {
		return command
	}
	return command
}

func slaveCommandForVersion(version string) []string {
	command := []string{
		fmt.Sprintf("--server-id=%d", defaultSlaveServerID),
		"--relay-log=relay-log",
		"--gtid-mode=ON",
		"--enforce-gtid-consistency=ON",
		"--skip-name-resolve=1",
		"--default-authentication-plugin=mysql_native_password",
	}
	if version == "v8.0" {
		return command
	}
	return command
}

func mysqlHealthCheck(rootPassword string) *runtime.HealthCheck {
	return &runtime.HealthCheck{
		Test: []string{
			"CMD",
			"mysqladmin",
			"ping",
			"-h",
			"127.0.0.1",
			"-uroot",
			"-p" + rootPassword,
		},
		Interval:    "5s",
		Timeout:     "5s",
		Retries:     30,
		StartPeriod: "20s",
	}
}

func masterInitSQL(replicationUser, replicationPassword string) string {
	return fmt.Sprintf(
		"-- create replication user\nCREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED WITH mysql_native_password BY '%s';\nGRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO '%s'@'%%';\nFLUSH PRIVILEGES;\n",
		replicationUser,
		replicationPassword,
		replicationUser,
	)
}

func slaveInitSQL(version, masterHost, replicationUser, replicationPassword string) string {
	if version == "v8.0" {
		return fmt.Sprintf(
			"-- configure replica channel\nSTOP REPLICA;\nRESET REPLICA ALL;\n\nCHANGE REPLICATION SOURCE TO\n  SOURCE_HOST='%s',\n  SOURCE_PORT=3306,\n  SOURCE_USER='%s',\n  SOURCE_PASSWORD='%s',\n  SOURCE_AUTO_POSITION=1,\n  GET_SOURCE_PUBLIC_KEY=1;\n\nSTART REPLICA;\n\nSET GLOBAL read_only = ON;\nSET GLOBAL super_read_only = ON;\n",
			masterHost,
			replicationUser,
			replicationPassword,
		)
	}
	return fmt.Sprintf(
		"-- configure slave channel\nSTOP SLAVE;\nRESET SLAVE ALL;\n\nCHANGE MASTER TO\n  MASTER_HOST='%s',\n  MASTER_PORT=3306,\n  MASTER_USER='%s',\n  MASTER_PASSWORD='%s',\n  MASTER_AUTO_POSITION=1;\n\nSTART SLAVE;\n\nSET GLOBAL read_only = ON;\nSET GLOBAL super_read_only = ON;\n",
		masterHost,
		replicationUser,
		replicationPassword,
	)
}

func masterSlaveBuildScript(name, envKeyPrefix, masterContainerName, slaveContainerName, slaveInitFile, version string) string {
	legacyPrimary := "SHOW SLAVE STATUS\\G"
	legacyFields := "Slave_IO_Running:|Slave_SQL_Running:|Seconds_Behind_Master:"
	primary := "SHOW REPLICA STATUS\\G"
	fields := "Replica_IO_Running:|Replica_SQL_Running:|Seconds_Behind_Source:"
	if version == "v5.7" {
		primary = legacyPrimary
		fields = legacyFields
	}

	return fmt.Sprintf(strings.TrimSpace(`
wait_mysql_healthy() {
    local name="$1"
    local retries=30
    for _ in $(seq 1 "$retries"); do
        status="$("$CONTAINER_ENGINE" inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || true)"
        if [ "$status" = "healthy" ]; then
            return 0
        fi
        sleep 2
    done
    echo "container $name did not become healthy" >&2
    return 1
}

echo "[mysql master-slave] waiting for %s master"
wait_mysql_healthy %s

echo "[mysql master-slave] waiting for %s slave"
wait_mysql_healthy %s

echo "[mysql master-slave] configuring replication for %s"
"$CONTAINER_ENGINE" exec -i %s mysql -uroot "-p${%s_ROOT_PASSWORD}" < %s

echo "[mysql master-slave] replication status for %s"
if ! "$CONTAINER_ENGINE" exec %s mysql -uroot "-p${%s_ROOT_PASSWORD}" -e %q | grep -E %q; then
    "$CONTAINER_ENGINE" exec %s mysql -uroot "-p${%s_ROOT_PASSWORD}" -e %q | grep -E %q || true
fi
`),
		name,
		masterContainerName,
		name,
		slaveContainerName,
		name,
		slaveContainerName,
		envKeyPrefix,
		slaveInitFile,
		name,
		slaveContainerName,
		envKeyPrefix,
		primary,
		fields,
		slaveContainerName,
		envKeyPrefix,
		legacyPrimary,
		legacyFields,
	)
}

func masterSlaveCheckScript(name, envKeyPrefix, masterContainerName, slaveContainerName string) string {
	return fmt.Sprintf(strings.TrimSpace(`
run_mysql_%s() {
    local container="$1"
    local sql="$2"
    "$CONTAINER_ENGINE" exec "$container" mysql -uroot "-p${%s_ROOT_PASSWORD}" -e "$sql"
}

echo "[mysql master-slave] container status for %s"
"$CONTAINER_ENGINE" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /%s|%s/'

echo "[mysql master-slave] replica status for %s"
if ! run_mysql_%s %s "SHOW REPLICA STATUS\\G" | grep -E "Replica_IO_Running:|Replica_SQL_Running:|Seconds_Behind_Source:|Last_IO_Error:|Last_SQL_Error:"; then
    run_mysql_%s %s "SHOW SLAVE STATUS\\G" | grep -E "Slave_IO_Running:|Slave_SQL_Running:|Seconds_Behind_Master:|Last_IO_Error:|Last_SQL_Error:"
fi

echo "[mysql master-slave] test replication for %s"
run_mysql_%s %s "CREATE DATABASE IF NOT EXISTS test_repl;"
sleep 2
run_mysql_%s %s "SHOW DATABASES;" | grep test_repl
`),
		envKeyPrefix,
		envKeyPrefix,
		name,
		regexpEscape(masterContainerName),
		regexpEscape(slaveContainerName),
		name,
		envKeyPrefix,
		slaveContainerName,
		envKeyPrefix,
		slaveContainerName,
		name,
		envKeyPrefix,
		masterContainerName,
		envKeyPrefix,
		slaveContainerName,
	)
}

func masterSlaveReadme(name, version, masterImage, slaveImage string, masterPort, slavePort int) string {
	return fmt.Sprintf(
		"# MySQL %s\n\n- template: master-slave\n- version: %s\n- master image: %s\n- slave image: %s\n- master port: %d\n- slave port: %d\n",
		name,
		version,
		masterImage,
		slaveImage,
		masterPort,
		slavePort,
	)
}

func regexpEscape(value string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		".", `\.`,
		"-", `\-`,
	)
	return replacer.Replace(value)
}
