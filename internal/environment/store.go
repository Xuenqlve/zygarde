package environment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

// Store persists environment metadata for lifecycle operations.
type Store interface {
	Save(env model.Environment) error
	Get(id string) (model.Environment, error)
	List() ([]model.Environment, error)
	SaveRuntimeArtifact(artifact runtime.RuntimeArtifact) error
	GetRuntimeArtifact(id string) (runtime.RuntimeArtifact, error)
}

// FileStore stores environments as JSON files under one local root directory.
type FileStore struct {
	rootDir string
}

// NewFileStore creates a file-backed environment store.
func NewFileStore(rootDir string) FileStore {
	return FileStore{rootDir: rootDir}
}

// Save writes one environment snapshot to local storage.
func (s FileStore) Save(env model.Environment) error {
	if env.ID == "" {
		return fmt.Errorf("environment id is required")
	}
	if err := os.MkdirAll(s.rootDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(s.rootDir, env.ID+".json")
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// SaveRuntimeArtifact writes one runtime artifact snapshot to local storage.
func (s FileStore) SaveRuntimeArtifact(artifact runtime.RuntimeArtifact) error {
	if artifact.EnvironmentID == "" {
		return fmt.Errorf("runtime artifact environment id is required")
	}
	if err := os.MkdirAll(s.rootDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(s.rootDir, artifact.EnvironmentID+".runtime.json")
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// Get loads one environment snapshot by id.
func (s FileStore) Get(id string) (model.Environment, error) {
	path := filepath.Join(s.rootDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Environment{}, err
	}
	var env model.Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return model.Environment{}, err
	}
	return env, nil
}

// List loads all persisted environment snapshots, sorted by update time descending.
func (s FileStore) List() ([]model.Environment, error) {
	entries, err := os.ReadDir(s.rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.Environment{}, nil
		}
		return nil, err
	}

	environments := make([]model.Environment, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".runtime.json") {
			continue
		}

		data, readErr := os.ReadFile(filepath.Join(s.rootDir, name))
		if readErr != nil {
			return nil, readErr
		}

		var env model.Environment
		if unmarshalErr := json.Unmarshal(data, &env); unmarshalErr != nil {
			return nil, unmarshalErr
		}
		environments = append(environments, env)
	}

	sort.Slice(environments, func(i, j int) bool {
		left := environments[i].UpdatedAt
		if left.IsZero() {
			left = environments[i].CreatedAt
		}
		right := environments[j].UpdatedAt
		if right.IsZero() {
			right = environments[j].CreatedAt
		}
		if left.Equal(right) {
			return environments[i].ID < environments[j].ID
		}
		return left.After(right)
	})

	return environments, nil
}

// GetRuntimeArtifact loads one runtime artifact snapshot by environment id.
func (s FileStore) GetRuntimeArtifact(id string) (runtime.RuntimeArtifact, error) {
	path := filepath.Join(s.rootDir, id+".runtime.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return runtime.RuntimeArtifact{}, err
	}
	var artifact runtime.RuntimeArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return runtime.RuntimeArtifact{}, err
	}
	return artifact, nil
}
