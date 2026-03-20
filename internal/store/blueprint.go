package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/xuenqlve/zygarde/internal/model"
)

// BlueprintFile captures one discovered blueprint file and its parsed metadata.
type BlueprintFile struct {
	Path      string
	Blueprint model.Blueprint
}

// BlueprintStore loads blueprint definitions from storage.
type BlueprintStore interface {
	LoadBlueprint(path string) (model.Blueprint, error)
	ListBlueprints(root string) ([]BlueprintFile, error)
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

// ListBlueprints scans one root directory for known blueprint file names.
func (s FileBlueprintStore) ListBlueprints(root string) ([]BlueprintFile, error) {
	if root == "" {
		root = "."
	}

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []BlueprintFile{}, nil
		}
		return nil, err
	}

	if !info.IsDir() {
		blueprint, err := s.LoadBlueprint(root)
		if err != nil {
			return nil, err
		}
		return []BlueprintFile{{
			Path:      filepath.Clean(root),
			Blueprint: blueprint,
		}}, nil
	}

	items := make([]BlueprintFile, 0)
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !isBlueprintCandidate(path) {
			return nil
		}

		blueprint, err := s.LoadBlueprint(path)
		if err != nil {
			return err
		}
		items = append(items, BlueprintFile{
			Path:      filepath.Clean(path),
			Blueprint: blueprint,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})
	return items, nil
}

func isBlueprintCandidate(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	switch base {
	case "zygarde.yaml", "zygarde.yml":
		return true
	}
	return strings.HasSuffix(base, ".blueprint.yaml") || strings.HasSuffix(base, ".blueprint.yml")
}
