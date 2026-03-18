package tool

import (
	"fmt"
	"net"
	"sync"
)

var (
	portDispenserMu          sync.Mutex
	portDispenserInitialized bool
	reservedPorts            map[int]struct{}
)

// InitPortDispenser initializes the task-scoped global port dispenser.
func InitPortDispenser() {
	portDispenserMu.Lock()
	defer portDispenserMu.Unlock()

	reservedPorts = make(map[int]struct{})
	portDispenserInitialized = true
}

// ResetPortDispenser clears the task-scoped global port dispenser.
func ResetPortDispenser() {
	portDispenserMu.Lock()
	defer portDispenserMu.Unlock()

	reservedPorts = nil
	portDispenserInitialized = false
}

// ReservePort validates and reserves one user-specified port for the current task.
func ReservePort(port int) error {
	portDispenserMu.Lock()
	defer portDispenserMu.Unlock()

	if err := ensureInitialized(); err != nil {
		return err
	}
	if err := validatePort(port); err != nil {
		return err
	}
	if _, exists := reservedPorts[port]; exists {
		return fmt.Errorf("port %d already reserved in current task", port)
	}
	if !isPortAvailable(port) {
		return fmt.Errorf("port %d is already in use", port)
	}

	reservedPorts[port] = struct{}{}
	return nil
}

// AllocatePort reserves the first available port starting from start for the current task.
func AllocatePort(start int) (int, error) {
	portDispenserMu.Lock()
	defer portDispenserMu.Unlock()

	if err := ensureInitialized(); err != nil {
		return 0, err
	}
	if err := validatePort(start); err != nil {
		return 0, err
	}

	for port := start; port <= 65535; port++ {
		if _, exists := reservedPorts[port]; exists {
			continue
		}
		if !isPortAvailable(port) {
			continue
		}
		reservedPorts[port] = struct{}{}
		return port, nil
	}

	return 0, fmt.Errorf("no available port found from %d", start)
}

func ensureInitialized() error {
	if !portDispenserInitialized {
		return fmt.Errorf("port dispenser is not initialized")
	}
	return nil
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d is out of range", port)
	}
	return nil
}

func isPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}
