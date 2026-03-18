package tidb

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
		Name:       "tidb-auto",
		Middleware: middlewareName,
		Template:   singleTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure tidb single: %v", err)
	}

	keys := []string{
		runtimecompose.ValuePDPort,
		runtimecompose.ValueTiKVPort,
		runtimecompose.ValueTiDBPort,
		runtimecompose.ValueTiDBStatusPort,
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
		Name:       "tidb-fixed",
		Middleware: middlewareName,
		Template:   singleTemplate,
		Values: map[string]any{
			runtimecompose.ValueTiDBPort: port,
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
		Name:       "tidb-cluster-auto",
		Middleware: middlewareName,
		Template:   clusterTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure tidb cluster: %v", err)
	}

	keys := []string{
		runtimecompose.ValuePD1Port,
		runtimecompose.ValuePD2Port,
		runtimecompose.ValuePD3Port,
		runtimecompose.ValueTiDB1Port,
		runtimecompose.ValueTiDB2Port,
		runtimecompose.ValueTiDB1StatusPort,
		runtimecompose.ValueTiDB2StatusPort,
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
		Name:       "tidb-cluster-fixed",
		Middleware: middlewareName,
		Template:   clusterTemplate,
		Values: map[string]any{
			runtimecompose.ValuePD1Port: 2379,
			runtimecompose.ValuePD2Port: 2379,
		},
	}, 1)
	if err == nil {
		t.Fatal("expected duplicate explicit tidb cluster port to be rejected")
	}
}
