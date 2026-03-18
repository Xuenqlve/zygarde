package postgresql

import (
	"fmt"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
	"github.com/xuenqlve/zygarde/internal/tool"
)

const (
	masterSlaveTemplate        = "master-slave"
	defaultMasterPort          = 5432
	defaultSlavePort           = 5433
	defaultReplicationUser     = "repl_user"
	defaultReplicationPassword = "repl_pass"
)

// NewMasterSlaveSpec returns the default PostgreSQL master-slave middleware spec.
func NewMasterSlaveSpec() tpl.Middleware {
	return &masterSlaveSpec{}
}

type masterSlaveSpec struct {
	services []model.BlueprintService
}

func (*masterSlaveSpec) Middleware() string { return middlewareName }
func (*masterSlaveSpec) Template() string   { return masterSlaveTemplate }
func (*masterSlaveSpec) IsDefault() bool    { return false }

func (s *masterSlaveSpec) Normalize(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	name := input.Name
	if name == "" {
		name = tpl.DefaultServiceName(s.Middleware(), index)
	}

	values := tpl.MergeValues(s.DefaultValues(), input.Values)
	version := defaultStringValue(values[runtimecompose.ValueVersion], defaultVersion)
	if err := validateVersion(version); err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize postgresql master-slave version: %w", err)
	}
	values[runtimecompose.ValueVersion] = version

	values[runtimecompose.ValueMasterServiceName] = defaultStringValue(values[runtimecompose.ValueMasterServiceName], fmt.Sprintf("%s-master", name))
	values[runtimecompose.ValueSlaveServiceName] = defaultStringValue(values[runtimecompose.ValueSlaveServiceName], fmt.Sprintf("%s-slave", name))
	values[runtimecompose.ValueMasterContainerName] = defaultStringValue(values[runtimecompose.ValueMasterContainerName], values[runtimecompose.ValueMasterServiceName].(string))
	values[runtimecompose.ValueSlaveContainerName] = defaultStringValue(values[runtimecompose.ValueSlaveContainerName], values[runtimecompose.ValueSlaveServiceName].(string))
	values[runtimecompose.ValueMasterImage] = defaultStringValue(values[runtimecompose.ValueMasterImage], imageForVersion(version))
	values[runtimecompose.ValueSlaveImage] = defaultStringValue(values[runtimecompose.ValueSlaveImage], imageForVersion(version))
	values[runtimecompose.ValueMasterDataDir] = defaultStringValue(values[runtimecompose.ValueMasterDataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueMasterServiceName].(string)))
	values[runtimecompose.ValueSlaveDataDir] = defaultStringValue(values[runtimecompose.ValueSlaveDataDir], fmt.Sprintf("./data/%s", values[runtimecompose.ValueSlaveServiceName].(string)))
	values[runtimecompose.ValueUser] = defaultStringValue(values[runtimecompose.ValueUser], "postgres")
	values[runtimecompose.ValuePassword] = defaultStringValue(values[runtimecompose.ValuePassword], "postgres123")
	values[runtimecompose.ValueDatabase] = defaultStringValue(values[runtimecompose.ValueDatabase], "app")
	values[runtimecompose.ValueReplicationUser] = defaultStringValue(values[runtimecompose.ValueReplicationUser], defaultReplicationUser)
	values[runtimecompose.ValueReplicationPassword] = defaultStringValue(values[runtimecompose.ValueReplicationPassword], defaultReplicationPassword)

	masterPort, err := allocateOrReservePort(values[runtimecompose.ValueMasterPort], hasValue(input.Values, runtimecompose.ValueMasterPort), defaultMasterPort, "postgresql master-slave master_port")
	if err != nil {
		return model.BlueprintService{}, err
	}
	values[runtimecompose.ValueMasterPort] = masterPort

	slavePort, err := allocateOrReservePort(values[runtimecompose.ValueSlavePort], hasValue(input.Values, runtimecompose.ValueSlavePort), defaultSlavePort, "postgresql master-slave slave_port")
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
		user := service.Values[runtimecompose.ValueUser].(string)
		password := service.Values[runtimecompose.ValuePassword].(string)
		database := service.Values[runtimecompose.ValueDatabase].(string)
		replUser := service.Values[runtimecompose.ValueReplicationUser].(string)
		replPassword := service.Values[runtimecompose.ValueReplicationPassword].(string)

		masterService := service.Values[runtimecompose.ValueMasterServiceName].(string)
		slaveService := service.Values[runtimecompose.ValueSlaveServiceName].(string)
		masterContainer := service.Values[runtimecompose.ValueMasterContainerName].(string)
		slaveContainer := service.Values[runtimecompose.ValueSlaveContainerName].(string)
		masterImage := service.Values[runtimecompose.ValueMasterImage].(string)
		slaveImage := service.Values[runtimecompose.ValueSlaveImage].(string)
		masterDataDir := service.Values[runtimecompose.ValueMasterDataDir].(string)
		slaveDataDir := service.Values[runtimecompose.ValueSlaveDataDir].(string)
		masterPort := service.Values[runtimecompose.ValueMasterPort].(int)
		slavePort := service.Values[runtimecompose.ValueSlavePort].(int)
		envKeyPrefix := serviceEnvKeyPrefix(service.Name)

		contexts = append(contexts,
			runtime.ComposeContext{
				EnvType:     runtimeType,
				ServiceName: masterService,
				Middleware:  service.Middleware,
				Template:    service.Template,
				Service: runtime.ServiceSpec{
					Image:         masterImage,
					ContainerName: masterContainer,
					Restart:       "unless-stopped",
					Environment: map[string]string{
						"POSTGRES_USER":     user,
						"POSTGRES_PASSWORD": password,
						"POSTGRES_DB":       database,
						"REPL_USER":         replUser,
						"REPL_PASSWORD":     replPassword,
					},
					Ports: []runtime.PortBinding{{HostPort: masterPort, ContainerPort: 5432, Protocol: "tcp"}},
					Volumes: []runtime.VolumeMount{
						{Source: masterDataDir, Target: "/var/lib/postgresql/data"},
						{Source: "./scripts/01-master-init.sh", Target: "/docker-entrypoint-initdb.d/01-master-init.sh", ReadOnly: true},
					},
					Command: []string{"postgres", "-c", "wal_level=replica", "-c", "max_wal_senders=10", "-c", "max_replication_slots=10"},
					HealthCheck: &runtime.HealthCheck{
						Test:        []string{"CMD-SHELL", fmt.Sprintf("pg_isready -U %s -d %s", user, database)},
						Interval:    "5s",
						Timeout:     "5s",
						Retries:     40,
						StartPeriod: "20s",
					},
				},
				Assets: []runtime.AssetSpec{
					{
						Name:    "postgres-master-slave-env",
						PathKey: "env_file",
						Content: fmt.Sprintf(
							"%s_VERSION=%s\n%s_POSTGRES_USER=%s\n%s_POSTGRES_PASSWORD=%s\n%s_POSTGRES_DB=%s\n%s_MASTER_PORT=%d\n%s_SLAVE_PORT=%d\n%s_REPL_USER=%s\n%s_REPL_PASSWORD=%s\n",
							envKeyPrefix, version,
							envKeyPrefix, user,
							envKeyPrefix, password,
							envKeyPrefix, database,
							envKeyPrefix, masterPort,
							envKeyPrefix, slavePort,
							envKeyPrefix, replUser,
							envKeyPrefix, replPassword,
						),
						Mode:      0o644,
						MergeMode: runtime.AssetMergeEnv,
					},
					{
						Name:      "postgres-master-slave-build",
						PathKey:   "build_script",
						Content:   postgresMasterSlaveBuildScript(service.Name, masterContainer, slaveContainer),
						Mode:      0o755,
						MergeMode: runtime.AssetMergeScript,
					},
					{
						Name:      "postgres-master-slave-check",
						PathKey:   "check_script",
						Content:   postgresMasterSlaveCheckScript(service.Name, envKeyPrefix, masterContainer, slaveContainer),
						Mode:      0o755,
						MergeMode: runtime.AssetMergeScript,
					},
					{
						Name:      "postgres-master-slave-readme",
						PathKey:   "readme_file",
						Content:   postgresMasterSlaveReadme(service.Name, version, masterImage, slaveImage, masterPort, slavePort, user, database),
						Mode:      0o644,
						MergeMode: runtime.AssetMergeReadme,
					},
					{
						Name:      "postgres-master-init",
						FileName:  "scripts/01-master-init.sh",
						Content:   postgresMasterInitScript(),
						Mode:      0o755,
						MergeMode: runtime.AssetMergeUnique,
					},
					{
						Name:      "postgres-start-slave",
						FileName:  "scripts/start-slave.sh",
						Content:   postgresStartSlaveScript(),
						Mode:      0o755,
						MergeMode: runtime.AssetMergeUnique,
					},
				},
				Metadata: tpl.MergeValues(nil, service.Values),
			},
			runtime.ComposeContext{
				EnvType:     runtimeType,
				ServiceName: slaveService,
				Middleware:  service.Middleware,
				Template:    service.Template,
				Service: runtime.ServiceSpec{
					Image:         slaveImage,
					ContainerName: slaveContainer,
					Restart:       "unless-stopped",
					Environment: map[string]string{
						"POSTGRES_USER":     user,
						"POSTGRES_PASSWORD": password,
						"REPL_USER":         replUser,
						"REPL_PASSWORD":     replPassword,
						"MASTER_HOST":       masterService,
						"MASTER_PORT":       "5432",
					},
					Ports: []runtime.PortBinding{{HostPort: slavePort, ContainerPort: 5432, Protocol: "tcp"}},
					Volumes: []runtime.VolumeMount{
						{Source: slaveDataDir, Target: "/var/lib/postgresql/data"},
						{Source: "./scripts/start-slave.sh", Target: "/scripts/start-slave.sh", ReadOnly: true},
					},
					Command: []string{"bash", "/scripts/start-slave.sh"},
					HealthCheck: &runtime.HealthCheck{
						Test:        []string{"CMD-SHELL", fmt.Sprintf("pg_isready -U %s", user)},
						Interval:    "5s",
						Timeout:     "5s",
						Retries:     40,
						StartPeriod: "25s",
					},
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
		return fmt.Errorf("postgresql master-slave validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != masterSlaveTemplate {
		return fmt.Errorf("postgresql master-slave validate: unexpected template %q", service.Template)
	}

	version, ok := service.Values[runtimecompose.ValueVersion].(string)
	if !ok || version == "" {
		return fmt.Errorf("postgresql master-slave validate version: must be a non-empty string")
	}
	if err := validateVersion(version); err != nil {
		return fmt.Errorf("postgresql master-slave validate version: %w", err)
	}

	fields := []struct{ key, name string }{
		{runtimecompose.ValueMasterServiceName, "master_service_name"},
		{runtimecompose.ValueSlaveServiceName, "slave_service_name"},
		{runtimecompose.ValueMasterContainerName, "master_container_name"},
		{runtimecompose.ValueSlaveContainerName, "slave_container_name"},
		{runtimecompose.ValueMasterImage, "master_image"},
		{runtimecompose.ValueSlaveImage, "slave_image"},
		{runtimecompose.ValueMasterDataDir, "master_data_dir"},
		{runtimecompose.ValueSlaveDataDir, "slave_data_dir"},
		{runtimecompose.ValueUser, "user"},
		{runtimecompose.ValuePassword, "password"},
		{runtimecompose.ValueDatabase, "database"},
		{runtimecompose.ValueReplicationUser, "replication_user"},
		{runtimecompose.ValueReplicationPassword, "replication_password"},
	}
	for _, field := range fields {
		value, ok := service.Values[field.key].(string)
		if !ok || value == "" {
			return fmt.Errorf("postgresql master-slave validate %s: must be a non-empty string", field.name)
		}
	}

	masterPort, err := normalizePort(service.Values[runtimecompose.ValueMasterPort])
	if err != nil {
		return fmt.Errorf("postgresql master-slave validate master_port: %w", err)
	}
	slavePort, err := normalizePort(service.Values[runtimecompose.ValueSlavePort])
	if err != nil {
		return fmt.Errorf("postgresql master-slave validate slave_port: %w", err)
	}
	if masterPort <= 0 || slavePort <= 0 {
		return fmt.Errorf("postgresql master-slave validate ports: must be greater than 0")
	}
	if masterPort == slavePort {
		return fmt.Errorf("postgresql master-slave validate ports: master_port and slave_port must be different")
	}

	return nil
}

func (*masterSlaveSpec) DefaultValues() map[string]any {
	return map[string]any{
		runtimecompose.ValueVersion:             defaultVersion,
		runtimecompose.ValueMasterImage:         "",
		runtimecompose.ValueSlaveImage:          "",
		runtimecompose.ValueMasterDataDir:       "",
		runtimecompose.ValueSlaveDataDir:        "",
		runtimecompose.ValueMasterPort:          defaultMasterPort,
		runtimecompose.ValueSlavePort:           defaultSlavePort,
		runtimecompose.ValueUser:                "postgres",
		runtimecompose.ValuePassword:            "postgres123",
		runtimecompose.ValueDatabase:            "app",
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

func postgresMasterInitScript() string {
	return `#!/usr/bin/env bash
set -euo pipefail

REPL_USER="${REPL_USER:-repl_user}"
REPL_PASSWORD="${REPL_PASSWORD:-repl_pass}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<SQL
DO
\$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${REPL_USER}') THEN
    EXECUTE format('CREATE ROLE %I WITH REPLICATION LOGIN PASSWORD %L', '${REPL_USER}', '${REPL_PASSWORD}');
  END IF;
END
\$\$;
SQL

if ! grep -q "host replication ${REPL_USER}" "$PGDATA/pg_hba.conf"; then
  echo "host replication ${REPL_USER} 0.0.0.0/0 md5" >> "$PGDATA/pg_hba.conf"
fi
`
}

func postgresStartSlaveScript() string {
	return `#!/usr/bin/env bash
set -euo pipefail

PGDATA="${PGDATA:-/var/lib/postgresql/data}"
REPL_USER="${REPL_USER:-repl_user}"
REPL_PASSWORD="${REPL_PASSWORD:-repl_pass}"
MASTER_HOST="${MASTER_HOST:-postgres-master}"
MASTER_PORT="${MASTER_PORT:-5432}"

if [ ! -s "$PGDATA/PG_VERSION" ]; then
  until pg_isready -h "$MASTER_HOST" -p "$MASTER_PORT" -U "${POSTGRES_USER:-postgres}" >/dev/null 2>&1; do
    sleep 2
  done

  rm -rf "$PGDATA"/*
  export PGPASSWORD="$REPL_PASSWORD"
  pg_basebackup -h "$MASTER_HOST" -p "$MASTER_PORT" -U "$REPL_USER" -D "$PGDATA" -Fp -Xs -R -P
  unset PGPASSWORD
  chmod 700 "$PGDATA"
fi

exec postgres -c hot_standby=on
`
}

func postgresMasterSlaveBuildScript(name, masterContainer, slaveContainer string) string {
	return fmt.Sprintf(`wait_healthy() {
  local name="$1"
  for _ in $(seq 1 60); do
    status="$("$CONTAINER_ENGINE" inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || true)"
    if [ "$status" = "healthy" ]; then return 0; fi
    sleep 2
  done
  return 1
}

echo "[postgresql master-slave] waiting for %s master"
wait_healthy %s

echo "[postgresql master-slave] waiting for %s slave"
wait_healthy %s
`, name, masterContainer, name, slaveContainer)
}

func postgresMasterSlaveCheckScript(name, envKeyPrefix, masterContainer, slaveContainer string) string {
	return fmt.Sprintf(`echo "[postgresql master-slave] container status for %s"
"$CONTAINER_ENGINE" ps --format 'table {{.Names}}\t{{.Status}}' | awk 'NR==1 || /%s|%s/'

echo "[postgresql master-slave] connectivity for %s"
"$CONTAINER_ENGINE" exec %s psql -U "${%s_USER}" -d postgres -c 'select 1;'
"$CONTAINER_ENGINE" exec %s psql -U "${%s_USER}" -d postgres -c 'select 1;'

echo "[postgresql master-slave] replication on master for %s"
ok=0
for _ in $(seq 1 60); do
  CNT="$("$CONTAINER_ENGINE" exec %s psql -U "${%s_USER}" -d postgres -tAc "select count(*) from pg_stat_replication;" | tr -d '[:space:]')"
  if [ "${CNT:-0}" -ge 1 ]; then ok=1; break; fi
  sleep 2
done
[ "$ok" -eq 1 ] || { echo "No replica found on master" >&2; exit 1; }
echo "replica_count=${CNT}"

echo "[postgresql master-slave] slave recovery mode for %s"
ok=0
for _ in $(seq 1 60); do
  REC="$("$CONTAINER_ENGINE" exec %s psql -U "${%s_USER}" -d postgres -tAc "select pg_is_in_recovery();" | tr -d '[:space:]')"
  if [ "$REC" = "t" ]; then ok=1; break; fi
  sleep 2
done
[ "$ok" -eq 1 ] || { echo "Slave is not in recovery mode" >&2; exit 1; }
echo "recovery_mode=${REC}"
`, name, masterContainer, slaveContainer, name,
		masterContainer, envKeyPrefix,
		slaveContainer, envKeyPrefix,
		name, masterContainer, envKeyPrefix,
		name, slaveContainer, envKeyPrefix)
}

func postgresMasterSlaveReadme(name, version, masterImage, slaveImage string, masterPort, slavePort int, user, database string) string {
	return fmt.Sprintf("# PostgreSQL %s\n\n- template: master-slave\n- version: %s\n- master image: %s\n- slave image: %s\n- master port: %d\n- slave port: %d\n- user: %s\n- database: %s\n", name, version, masterImage, slaveImage, masterPort, slavePort, user, database)
}
