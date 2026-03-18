package config

import (
	"os"

	"github.com/xuenqlve/zygarde/internal/runtime"
)

const defaultContainerEngine = "docker"

// Config contains minimal platform defaults for the create flow.
type Config struct {
	DefaultEnvironmentType runtime.EnvironmentType
	ContainerEngine        string
}

// Default returns the minimal runtime defaults used by the application.
func Default() Config {
	containerEngine := os.Getenv("ZYGARDE_CONTAINER_ENGINE")
	if containerEngine == "" {
		containerEngine = defaultContainerEngine
	}

	return Config{
		DefaultEnvironmentType: runtime.EnvironmentTypeCompose,
		ContainerEngine:        containerEngine,
	}
}
