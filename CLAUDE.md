# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Zygarde is a modern, modular environment setup and deployment tool that maintains "order" and "integrity" in development environments. It provides a declarative, developer-friendly solution for one-click deployment of local database environments using container orchestration.

## Architecture

The project follows a modular design with five core components:

1. **Template Manager** - Handles template CRUD operations, parsing, and validation of container templates with variables like `{{ .Port }}`
2. **Blueprint Manager** - Orchestrates multiple templates into deployable blueprints, manages variable assignments and template rendering
3. **Environment Manager** - Manages environment instances, tracks states (Creating, Running, Stopped, Error), and maintains metadata
4. **Deployment Engine** - Executes Docker Compose commands with project isolation using `-p` parameter to avoid naming conflicts
5. **Coordinator** - Unified facade that orchestrates all components and provides transactional guarantees

## Development Commands

Since this is an early-stage Go project without established build tools yet:

```bash
# Build the application
go build -o zygarde ./cmd/main.go

# Run the application (requires -config parameter)
go run ./cmd/main.go -config <config-file>

# Format code
go fmt ./...

# Run tests (when available)
go test ./...

# Run tests for specific package
go test ./internal/config

# Get dependencies
go mod tidy

# Vendor dependencies
go mod vendor
```

## Configuration System

The project uses a flexible configuration system with factory pattern:

- Supports multiple config types: `file` (default), `nacos`, `etcd`
- Configuration is loaded via `-config` flag pointing to a config file
- The config system uses a registry pattern for different config sources
- Logging configuration supports debug/info/warn levels with both console and file output

## Code Architecture Patterns

### Plugin/Factory Pattern
The codebase extensively uses registry-based factory patterns for:
- **Config System** (`internal/config/base.go`): `RegisterConfig()` and `GetConfig()` for pluggable configuration sources
- **Data Source System** (`internal/data_source/data_source.go`): `RegisterPlugin()` and `GetDataSource()` for pluggable data sources

### Module Structure
```
/cmd/           - Application entry points
/internal/      - Private application code
  /config/      - Configuration management with pluggable backends
  /data_source/ - Data source abstraction with plugin system  
  /log/         - Centralized logging using zerolog
/pkg/           - Public library code (future external APIs)
```

### Interface Design
- `Config` interface for configuration backends
- `DataSource` interface for data source plugins
- Singleton vs non-singleton plugin registration supported

## Development Guidelines from .cursorrules

- Follow Go standard project layout
- Each core module should be an independent package with clear boundaries
- Use interfaces for module interactions to maintain loose coupling
- Implement dependency injection for testability
- Every exported function and type must have documentation comments
- Package comments are required for all packages
- Use conventional commits format for version control
- Table-driven tests for parameterized testing
- Unit test coverage target: 80%

## Important Notes

- The project is in early development stage - main.go is currently empty
- Uses zerolog for structured logging with both console and file output
- Configuration requires a `-config` parameter - the application will panic without it
- Plugin systems use reflection for non-singleton instances
- Error handling follows Go standard patterns with meaningful error messages
- See TODO.md for current development roadmap and progress tracking

## Template System (Planned)

When implementing templates:
- Support Go template syntax like `{{ .Port }}`
- Extract and validate template variables from metadata
- Provide template inheritance and composition capabilities
- Prevent template injection attacks through input validation