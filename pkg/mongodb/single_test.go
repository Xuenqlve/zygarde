package mongodb

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
		Name:       "mongodb-auto",
		Middleware: middlewareName,
		Template:   singleTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure mongodb single: %v", err)
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
		Name:       "mongodb-fixed",
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

func TestShardedSpecConfigureAllocatesPortWhenMissing(t *testing.T) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	releaseDefault := occupyDefaultPortIfPossible(t)
	if releaseDefault != nil {
		defer releaseDefault()
	}

	spec := NewShardedSpec()
	service, err := spec.Configure(tpl.ServiceInput{
		Name:       "mongodb-sharded",
		Middleware: middlewareName,
		Template:   shardedTemplate,
	}, 1)
	if err != nil {
		t.Fatalf("configure mongodb sharded: %v", err)
	}

	port, err := normalizePort(service.Values[runtimecompose.ValueMongosPort])
	if err != nil {
		t.Fatalf("normalize configured mongos port: %v", err)
	}
	if port < defaultMongosPort {
		t.Fatalf("expected allocated mongos port >= %d, got %d", defaultMongosPort, port)
	}
}

func TestShardedSpecConfigureRejectsOccupiedExplicitPort(t *testing.T) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy explicit port: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	spec := NewShardedSpec()
	_, err = spec.Configure(tpl.ServiceInput{
		Name:       "mongodb-sharded",
		Middleware: middlewareName,
		Template:   shardedTemplate,
		Values: map[string]any{
			runtimecompose.ValueMongosPort: port,
		},
	}, 1)
	if err == nil {
		t.Fatalf("expected occupied explicit port %d to be rejected", port)
	}
}

func TestReplicaSetSpecConfigureRejectsDuplicatePorts(t *testing.T) {
	tool.InitPortDispenser()
	defer tool.ResetPortDispenser()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy explicit port: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	spec := NewReplicaSetSpec()
	_, err = spec.Configure(tpl.ServiceInput{
		Name:       "mongodb-rs",
		Middleware: middlewareName,
		Template:   replicaSetTemplate,
		Values: map[string]any{
			runtimecompose.ValueRS1Port: port,
			runtimecompose.ValueRS2Port: port,
		},
	}, 1)
	if err == nil {
		t.Fatalf("expected duplicate replica-set port %d to be rejected", port)
	}
}

func occupyDefaultPortIfPossible(t *testing.T) func() {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:27017")
	if err != nil {
		return nil
	}
	return func() {
		_ = listener.Close()
	}
}
