package config

import "github.com/xuenqlve/zygarde/internal/runtime"

// Config contains minimal platform defaults for the create flow.
type Config struct {
	DefaultEnvironmentType runtime.EnvironmentType
}

// Default returns the minimal runtime defaults used by the application.
func Default() Config {
	return Config{
		DefaultEnvironmentType: runtime.EnvironmentTypeCompose,
	}
}
