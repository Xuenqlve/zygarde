package model

import "time"

// EnvironmentStatus represents the lifecycle state of an environment.
type EnvironmentStatus string

const (
	EnvironmentStatusPending    EnvironmentStatus = "pending"
	EnvironmentStatusCreating   EnvironmentStatus = "creating"
	EnvironmentStatusRunning    EnvironmentStatus = "running"
	EnvironmentStatusStopping   EnvironmentStatus = "stopping"
	EnvironmentStatusStopped    EnvironmentStatus = "stopped"
	EnvironmentStatusDestroying EnvironmentStatus = "destroying"
	EnvironmentStatusDestroyed  EnvironmentStatus = "destroyed"
	EnvironmentStatusError      EnvironmentStatus = "error"
)

// Environment describes a runtime environment created from a blueprint.
type Environment struct {
	ID               string
	Name             string
	BlueprintName    string
	BlueprintVersion string
	RuntimeType      string
	Status           EnvironmentStatus
	ProjectName      string
	WorkspaceDir     string
	ComposeFile      string
	Endpoints        []Endpoint
	CreatedAt        time.Time
	UpdatedAt        time.Time
	LastError        string
}

// Endpoint describes a reachable service endpoint exposed by an environment.
type Endpoint struct {
	Name     string
	Host     string
	Port     int
	Protocol string
}
