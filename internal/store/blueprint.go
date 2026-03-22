package store

import (
	"fmt"
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
	ResolveBlueprint(ref string, root string) (BlueprintFile, error)
	SaveBlueprint(path string, blueprint model.Blueprint) error
	UpdateBlueprint(path string, blueprint model.Blueprint) error
	DeleteBlueprint(path string) error
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

// ResolveBlueprint resolves one blueprint by file path or blueprint name.
func (s FileBlueprintStore) ResolveBlueprint(ref string, root string) (BlueprintFile, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return BlueprintFile{}, fmt.Errorf("blueprint reference is required")
	}

	if info, err := os.Stat(ref); err == nil {
		if info.IsDir() {
			return BlueprintFile{}, fmt.Errorf("blueprint path is a directory: %s", ref)
		}
		blueprint, loadErr := s.LoadBlueprint(ref)
		if loadErr != nil {
			return BlueprintFile{}, loadErr
		}
		return BlueprintFile{
			Path:      filepath.Clean(ref),
			Blueprint: blueprint,
		}, nil
	} else if !os.IsNotExist(err) {
		return BlueprintFile{}, err
	}

	if looksLikeBlueprintPath(ref) {
		return BlueprintFile{}, fmt.Errorf("blueprint file not found: %s", ref)
	}

	items, err := s.ListBlueprints(root)
	if err != nil {
		return BlueprintFile{}, err
	}

	matches := make([]BlueprintFile, 0, 1)
	for _, item := range items {
		if item.Blueprint.Name == ref {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return BlueprintFile{}, fmt.Errorf("blueprint not found by name: %s", ref)
	case 1:
		return matches[0], nil
	default:
		paths := make([]string, 0, len(matches))
		for _, item := range matches {
			paths = append(paths, item.Path)
		}
		return BlueprintFile{}, fmt.Errorf("blueprint name %s is ambiguous: %s", ref, strings.Join(paths, ", "))
	}
}

// SaveBlueprint writes one blueprint YAML file.
func (FileBlueprintStore) SaveBlueprint(path string, blueprint model.Blueprint) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return fmt.Errorf("blueprint path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("blueprint file already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}

	data, err := yaml.Marshal(&blueprint)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// UpdateBlueprint rewrites one existing blueprint YAML file.
func (FileBlueprintStore) UpdateBlueprint(path string, blueprint model.Blueprint) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return fmt.Errorf("blueprint path is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("blueprint path is a directory: %s", path)
	}
	data, err := yaml.Marshal(&blueprint)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// DeleteBlueprint removes one blueprint file.
func (FileBlueprintStore) DeleteBlueprint(path string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return fmt.Errorf("blueprint path is required")
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return nil
}

func isBlueprintCandidate(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	switch base {
	case "zygarde.yaml", "zygarde.yml":
		return true
	}
	return strings.HasSuffix(base, ".blueprint.yaml") || strings.HasSuffix(base, ".blueprint.yml")
}

func looksLikeBlueprintPath(ref string) bool {
	base := strings.ToLower(filepath.Base(ref))
	return strings.Contains(ref, string(filepath.Separator)) ||
		strings.HasSuffix(base, ".yaml") ||
		strings.HasSuffix(base, ".yml")
}
