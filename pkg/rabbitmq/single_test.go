package rabbitmq

import (
	"net"
	"testing"

	runtimecompose "github.com/xuenqlve/zygarde/internal/runtime/compose"
	tpl "github.com/xuenqlve/zygarde/internal/template"
	"github.com/xuenqlve/zygarde/internal/tool"
)

func TestSingleSpecConfigureAllocatesPortsWhenMissing(t *testing.T) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	spec := NewSingleSpec()
	service, err := spec.Configure(tpl.ServiceInput{
		Name:       "rabbitmq-auto",
		Middleware: middlewareName,
		Template:   singleTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure rabbitmq single: %v", err)
	}

	amqpPort, err := normalizePort(service.Values[runtimecompose.ValueAMQPPort])
	if err != nil {
		t.Fatalf("normalize configured amqp port: %v", err)
	}
	if amqpPort < defaultAMQPPort {
		t.Fatalf("expected allocated amqp port >= %d, got %d", defaultAMQPPort, amqpPort)
	}

	managementPort, err := normalizePort(service.Values[runtimecompose.ValueManagementPort])
	if err != nil {
		t.Fatalf("normalize configured management port: %v", err)
	}
	if managementPort <= 0 {
		t.Fatalf("expected allocated management port > 0, got %d", managementPort)
	}
	if amqpPort == managementPort {
		t.Fatalf("expected amqp and management ports to be different, got %d", amqpPort)
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
		Name:       "rabbitmq-fixed",
		Middleware: middlewareName,
		Template:   singleTemplate,
		Values: map[string]any{
			runtimecompose.ValueAMQPPort: port,
		},
	}, 1)
	if err == nil {
		t.Fatalf("expected occupied explicit port %d to be rejected", port)
	}
}

func TestClusterSpecConfigureAllocatesPortsWhenMissing(t *testing.T) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	spec := NewClusterSpec()
	service, err := spec.Configure(tpl.ServiceInput{
		Name:       "rabbitmq-cluster-auto",
		Middleware: middlewareName,
		Template:   clusterTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure rabbitmq cluster: %v", err)
	}

	keys := []string{
		runtimecompose.ValueRabbit1AMQPPort,
		runtimecompose.ValueRabbit2AMQPPort,
		runtimecompose.ValueRabbit3AMQPPort,
		runtimecompose.ValueRabbit1ManagementPort,
		runtimecompose.ValueRabbit2ManagementPort,
		runtimecompose.ValueRabbit3ManagementPort,
	}
	seen := make(map[int]struct{}, len(keys))
	for _, key := range keys {
		port, err := normalizePort(service.Values[key])
		if err != nil {
			t.Fatalf("normalize %s: %v", key, err)
		}
		if port <= 0 {
			t.Fatalf("expected %s > 0, got %d", key, port)
		}
		if _, ok := seen[port]; ok {
			t.Fatalf("expected unique allocated port for %s, got duplicate %d", key, port)
		}
		seen[port] = struct{}{}
	}
}

func TestClusterSpecConfigureRejectsDuplicateExplicitPort(t *testing.T) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	spec := NewClusterSpec()
	_, err := spec.Configure(tpl.ServiceInput{
		Name:       "rabbitmq-cluster-fixed",
		Middleware: middlewareName,
		Template:   clusterTemplate,
		Values: map[string]any{
			runtimecompose.ValueRabbit1AMQPPort: 5672,
			runtimecompose.ValueRabbit2AMQPPort: 5672,
		},
	}, 1)
	if err == nil {
		t.Fatal("expected duplicate explicit rabbitmq cluster port to be rejected")
	}
}
