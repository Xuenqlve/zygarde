package clickhouse

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
		Name:       "clickhouse-auto",
		Middleware: middlewareName,
		Template:   singleTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure clickhouse single: %v", err)
	}

	httpPort, err := normalizePort(service.Values[runtimecompose.ValueHTTPPort])
	if err != nil {
		t.Fatalf("normalize configured http port: %v", err)
	}
	tcpPort, err := normalizePort(service.Values[runtimecompose.ValueTCPPort])
	if err != nil {
		t.Fatalf("normalize configured tcp port: %v", err)
	}
	if httpPort <= 0 || tcpPort <= 0 {
		t.Fatalf("expected allocated ports > 0, got http=%d tcp=%d", httpPort, tcpPort)
	}
	if httpPort == tcpPort {
		t.Fatalf("expected different allocated ports, got http=%d tcp=%d", httpPort, tcpPort)
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
		Name:       "clickhouse-fixed",
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
		Name:       "clickhouse-cluster-auto",
		Middleware: middlewareName,
		Template:   clusterTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure clickhouse cluster: %v", err)
	}

	keys := []string{
		runtimecompose.ValueCH1HTTPPort,
		runtimecompose.ValueCH2HTTPPort,
		runtimecompose.ValueCH3HTTPPort,
		runtimecompose.ValueCH1TCPPort,
		runtimecompose.ValueCH2TCPPort,
		runtimecompose.ValueCH3TCPPort,
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
		Name:       "clickhouse-cluster-fixed",
		Middleware: middlewareName,
		Template:   clusterTemplate,
		Values: map[string]any{
			runtimecompose.ValueCH1HTTPPort: 8123,
			runtimecompose.ValueCH2HTTPPort: 8123,
		},
	}, 1)
	if err == nil {
		t.Fatal("expected duplicate explicit clickhouse cluster port to be rejected")
	}
}
