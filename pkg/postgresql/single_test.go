package postgresql

import (
	"net"
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
		Name:       "postgres-auto",
		Middleware: middlewareName,
		Template:   singleTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure postgresql single: %v", err)
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
		Name:       "postgres-fixed",
		Middleware: middlewareName,
		Template:   singleTemplate,
		Values: map[string]any{
			runtimecompose.ValuePort: port,
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
		Name:       "postgres-ms",
		Middleware: middlewareName,
		Template:   masterSlaveTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure postgresql master-slave: %v", err)
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

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy explicit port: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	spec := NewMasterSlaveSpec()
	_, err = spec.Configure(tpl.ServiceInput{
		Name:       "postgres-ms",
		Middleware: middlewareName,
		Template:   masterSlaveTemplate,
		Values: map[string]any{
			runtimecompose.ValueMasterPort: port,
			runtimecompose.ValueSlavePort:  port,
		},
	}, 1)
	if err == nil {
		t.Fatalf("expected duplicate master/slave port %d to be rejected", port)
	}
}

func occupyDefaultPortIfPossible(t *testing.T) func() {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:5432")
	if err != nil {
		return nil
	}
	return func() {
		_ = listener.Close()
	}
}
