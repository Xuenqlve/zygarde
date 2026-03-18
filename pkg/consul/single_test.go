package consul

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
		Name:       "consul-auto",
		Middleware: middlewareName,
		Template:   singleTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure consul single: %v", err)
	}

	httpPort, err := normalizePort(service.Values[runtimecompose.ValueHTTPPort])
	if err != nil {
		t.Fatalf("normalize configured http port: %v", err)
	}
	dnsPort, err := normalizePort(service.Values[runtimecompose.ValueDNSPort])
	if err != nil {
		t.Fatalf("normalize configured dns port: %v", err)
	}
	serverPort, err := normalizePort(service.Values[runtimecompose.ValueServerPort])
	if err != nil {
		t.Fatalf("normalize configured server port: %v", err)
	}
	if httpPort <= 0 || dnsPort <= 0 || serverPort <= 0 {
		t.Fatalf("expected allocated ports > 0, got http=%d dns=%d server=%d", httpPort, dnsPort, serverPort)
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
		Name:       "consul-fixed",
		Middleware: middlewareName,
		Template:   singleTemplate,
		Values: map[string]any{
			runtimecompose.ValueHTTPPort: port,
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
		Name:       "consul-cluster-auto",
		Middleware: middlewareName,
		Template:   clusterTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure consul cluster: %v", err)
	}

	keys := []string{
		runtimecompose.ValueConsul1HTTPPort,
		runtimecompose.ValueConsul1DNSPort,
		runtimecompose.ValueConsul2HTTPPort,
		runtimecompose.ValueConsul3HTTPPort,
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
		Name:       "consul-cluster-fixed",
		Middleware: middlewareName,
		Template:   clusterTemplate,
		Values: map[string]any{
			runtimecompose.ValueConsul1HTTPPort: 8500,
			runtimecompose.ValueConsul2HTTPPort: 8500,
		},
	}, 1)
	if err == nil {
		t.Fatal("expected duplicate explicit consul cluster port to be rejected")
	}
}
