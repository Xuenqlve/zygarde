package environment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const currentEnvironmentFile = ".zygarde/current-environment"

// CurrentEnvironment stores the current working directory environment pointer.
type CurrentEnvironment struct {
	EnvironmentID string `json:"environment_id"`
	WorkspaceDir  string `json:"workspace_dir"`
	ProjectName   string `json:"project_name"`
}

// SaveCurrent writes the current working directory environment marker.
func SaveCurrent(ref CurrentEnvironment) error {
	if ref.EnvironmentID == "" {
		return fmt.Errorf("current environment id is required")
	}

	path, err := currentEnvironmentPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(ref, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// LoadCurrent reads the current working directory environment marker.
func LoadCurrent() (CurrentEnvironment, error) {
	path, err := currentEnvironmentPath()
	if err != nil {
		return CurrentEnvironment{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return CurrentEnvironment{}, err
	}

	var ref CurrentEnvironment
	if err := json.Unmarshal(data, &ref); err != nil {
		return CurrentEnvironment{}, err
	}
	if ref.EnvironmentID == "" {
		return CurrentEnvironment{}, fmt.Errorf("current environment id is required")
	}
	return ref, nil
}

// ClearCurrent removes the current working directory environment marker.
func ClearCurrent() error {
	path, err := currentEnvironmentPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func currentEnvironmentPath() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, currentEnvironmentFile), nil
}
