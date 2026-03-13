package store

import (
	"os"

	"gopkg.in/yaml.v2"

	"github.com/xuenqlve/zygarde/internal/model"
)

// BlueprintStore loads blueprint definitions from storage.
type BlueprintStore interface {
	LoadBlueprint(path string) (model.Blueprint, error)
}

// FileBlueprintStore loads blueprint definitions from local files.
type FileBlueprintStore struct{}

// NewFileBlueprintStore creates a file-based blueprint store.
func NewFileBlueprintStore() FileBlueprintStore {
	return FileBlueprintStore{}
}

// LoadBlueprint reads and unmarshals one blueprint YAML file.
func (FileBlueprintStore) LoadBlueprint(path string) (model.Blueprint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Blueprint{}, err
	}

	var blueprint model.Blueprint
	if err := yaml.Unmarshal(data, &blueprint); err != nil {
		return model.Blueprint{}, err
	}

	return blueprint, nil
}
