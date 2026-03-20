package mysql

import (
	"net"
	"strings"
	"testing"

	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
	"github.com/xuenqlve/zygarde/internal/tool"
)

func TestSingleSpecConfigureAllocatesPortWhenMissing(t *testing.T) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	releaseDefault := occupyDefaultPortIfPossible(t)
	if releaseDefault != nil {
		defer releaseDefault()
	}

	spec := NewSingleSpec()
	service, err := spec.Configure(tpl.ServiceInput{
		Name:       "mysql-auto",
		Middleware: middlewareName,
		Template:   singleTemplate,
		Values: map[string]any{
			runtimecompose.ValueRootPassword: "root",
		},
	}, 1)
	if err != nil {
		t.Fatalf("configure mysql single: %v", err)
	}

	port, err := normalizePort(service.Values[runtimecompose.ValuePort])
	if err != nil {
		t.Fatalf("normalize configured port: %v", err)
	}
	if port < defaultPort {
		t.Fatalf("expected allocated port >= %d, got %d", defaultPort, port)
	}
}

func TestSingleSpecConfigureRejectsOccupiedExplicitPort(t *testing.T) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy explicit port: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	spec := NewSingleSpec()
	_, err = spec.Configure(tpl.ServiceInput{
		Name:       "mysql-fixed",
		Middleware: middlewareName,
		Template:   singleTemplate,
		Values: map[string]any{
			runtimecompose.ValuePort:         port,
			runtimecompose.ValueRootPassword: "root",
		},
	}, 1)
	if err == nil {
		t.Fatalf("expected occupied explicit port %d to be rejected", port)
	}
}

func TestMasterSlaveSpecConfigureAllocatesPortsWhenMissing(t *testing.T) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	releaseDefault := occupyDefaultPortIfPossible(t)
	if releaseDefault != nil {
		defer releaseDefault()
	}

	spec := NewMasterSlaveSpec()
	service, err := spec.Configure(tpl.ServiceInput{
		Name:       "mysql-ms",
		Middleware: middlewareName,
		Template:   masterSlaveTemplate,
		Values: map[string]any{
			runtimecompose.ValueRootPassword: "root",
		},
	}, 1)
	if err != nil {
		t.Fatalf("configure mysql master-slave: %v", err)
	}

	masterPort, err := normalizePort(service.Values[runtimecompose.ValueMasterPort])
	if err != nil {
		t.Fatalf("normalize configured master port: %v", err)
	}
	if masterPort < defaultMasterPort {
		t.Fatalf("expected allocated master port >= %d, got %d", defaultMasterPort, masterPort)
	}

	slavePort, err := normalizePort(service.Values[runtimecompose.ValueSlavePort])
	if err != nil {
		t.Fatalf("normalize configured slave port: %v", err)
	}
	if slavePort <= 0 {
		t.Fatalf("expected allocated slave port > 0, got %d", slavePort)
	}
	if masterPort == slavePort {
		t.Fatalf("expected master and slave ports to be different, got %d", masterPort)
	}
}

func TestMasterSlaveSpecConfigureRejectsDuplicatePorts(t *testing.T) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	port := freeEphemeralPort(t)

	spec := NewMasterSlaveSpec()
	_, err := spec.Configure(tpl.ServiceInput{
		Name:       "mysql-ms",
		Middleware: middlewareName,
		Template:   masterSlaveTemplate,
		Values: map[string]any{
			runtimecompose.ValueMasterPort:      port,
			runtimecompose.ValueSlavePort:       port,
			runtimecompose.ValueRootPassword:    "root",
			runtimecompose.ValueVersion:         "v8.0",
			runtimecompose.ValueReplicationUser: "repl",
		},
	}, 1)
	if err == nil {
		t.Fatalf("expected duplicate master/slave port %d to be rejected", port)
	}
}

func TestMasterSlaveCheckScriptUsesVersionSpecificReplicaStatusCommand(t *testing.T) {
	check57 := masterSlaveCheckScript("mysql-ms", "MYSQL_MS", "mysql-ms-master", "mysql-ms-slave", "v5.7")
	if !strings.Contains(check57, `"SHOW SLAVE STATUS\\G"`) {
		t.Fatalf("expected v5.7 check script to use SHOW SLAVE STATUS, got %q", check57)
	}
	if strings.Contains(check57, `"SHOW REPLICA STATUS\\G"`) {
		t.Fatalf("did not expect v5.7 check script to use SHOW REPLICA STATUS, got %q", check57)
	}

	check80 := masterSlaveCheckScript("mysql-ms", "MYSQL_MS", "mysql-ms-master", "mysql-ms-slave", "v8.0")
	if !strings.Contains(check80, `"SHOW REPLICA STATUS\\G"`) {
		t.Fatalf("expected v8.0 check script to use SHOW REPLICA STATUS, got %q", check80)
	}
}

func TestMasterSlaveBuildScriptWaitsForVersionSpecificReplicationReady(t *testing.T) {
	build57 := masterSlaveBuildScript("mysql-ms", "MYSQL_MS", "mysql-ms-master", "mysql-ms-slave", "slave-init.sql", "v5.7")
	if !strings.Contains(build57, "wait_replication_ready") {
		t.Fatalf("expected build script to wait for replication readiness, got %q", build57)
	}
	if !strings.Contains(build57, "Slave_IO_Running: Yes") || !strings.Contains(build57, "Slave_SQL_Running: Yes") {
		t.Fatalf("expected v5.7 build script to wait on slave readiness fields, got %q", build57)
	}

	build80 := masterSlaveBuildScript("mysql-ms", "MYSQL_MS", "mysql-ms-master", "mysql-ms-slave", "slave-init.sql", "v8.0")
	if !strings.Contains(build80, "Replica_IO_Running: Yes") || !strings.Contains(build80, "Replica_SQL_Running: Yes") {
		t.Fatalf("expected v8.0 build script to wait on replica readiness fields, got %q", build80)
	}
}

func occupyDefaultPortIfPossible(t *testing.T) func() {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:3306")
	if err != nil {
		return nil
	}
	return func() {
		_ = listener.Close()
	}
}

func freeEphemeralPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate ephemeral port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
