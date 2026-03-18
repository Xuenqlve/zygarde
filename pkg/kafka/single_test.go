package kafka

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

	spec := NewSingleSpec()
	service, err := spec.Configure(tpl.ServiceInput{
		Name:       "kafka-auto",
		Middleware: middlewareName,
		Template:   singleTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure kafka single: %v", err)
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
		Name:       "kafka-fixed",
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

func TestClusterSpecConfigureAllocatesPortsWhenMissing(t *testing.T) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	spec := NewClusterSpec()
	service, err := spec.Configure(tpl.ServiceInput{
		Name:       "kafka-cluster-auto",
		Middleware: middlewareName,
		Template:   clusterTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure kafka cluster: %v", err)
	}

	keys := []string{
		runtimecompose.ValueKafka1Port,
		runtimecompose.ValueKafka2Port,
		runtimecompose.ValueKafka3Port,
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
		Name:       "kafka-cluster-fixed",
		Middleware: middlewareName,
		Template:   clusterTemplate,
		Values: map[string]any{
			runtimecompose.ValueKafka1Port: 9092,
			runtimecompose.ValueKafka2Port: 9092,
		},
	}, 1)
	if err == nil {
		t.Fatal("expected duplicate explicit kafka cluster port to be rejected")
	}
}
