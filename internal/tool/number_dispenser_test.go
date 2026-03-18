package tool

import (
	"net"
	"testing"
)

func TestAllocatePortSkipsReservedPort(t *testing.T) {
	InitPortDispenser()
	defer ResetPortDispenser()

	first, err := AllocatePort(20000)
	if err != nil {
		t.Fatalf("allocate first port: %v", err)
	}
	second, err := AllocatePort(first)
	if err != nil {
		t.Fatalf("allocate second port: %v", err)
	}
	if second == first {
		t.Fatalf("expected second port to differ from first, got %d", second)
	}
}

func TestAllocatePortSkipsOccupiedPort(t *testing.T) {
	occupied := occupyPort(t)
	InitPortDispenser()
	defer ResetPortDispenser()

	allocated, err := AllocatePort(occupied)
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	if allocated == occupied {
		t.Fatalf("expected occupied port %d to be skipped", occupied)
	}
}

func TestReservePortRejectsOccupiedPort(t *testing.T) {
	occupied := occupyPort(t)
	InitPortDispenser()
	defer ResetPortDispenser()

	if err := ReservePort(occupied); err == nil {
		t.Fatalf("expected occupied port %d to be rejected", occupied)
	}
}

func TestResetPortDispenserClearsState(t *testing.T) {
	InitPortDispenser()

	first, err := AllocatePort(21000)
	if err != nil {
		t.Fatalf("allocate first port: %v", err)
	}

	ResetPortDispenser()
	InitPortDispenser()
	defer ResetPortDispenser()

	if err := ReservePort(first); err != nil {
		t.Fatalf("expected port %d to be reusable after reset: %v", first, err)
	}
}

func occupyPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy port: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})
	return listener.Addr().(*net.TCPAddr).Port
}
