package coordinator

import (
	"context"
	"sort"
	"time"

	"github.com/xuenqlve/zygarde/internal/model"
)

// ListItem captures one environment summary for list output.
type ListItem struct {
	ID            string
	Name          string
	BlueprintName string
	RuntimeType   string
	Status        model.EnvironmentStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Endpoints     []model.Endpoint
}

// ListResult captures all persisted environments.
type ListResult struct {
	Items []ListItem
}

// List returns all persisted environments from local storage.
func (c Coordinator) List(_ context.Context) (*ListResult, error) {
	environments, err := c.environments.List()
	if err != nil {
		return nil, err
	}

	items := make([]ListItem, 0, len(environments))
	for _, env := range environments {
		items = append(items, ListItem{
			ID:            env.ID,
			Name:          env.Name,
			BlueprintName: env.BlueprintName,
			RuntimeType:   env.RuntimeType,
			Status:        env.Status,
			CreatedAt:     env.CreatedAt,
			UpdatedAt:     env.UpdatedAt,
			Endpoints:     env.Endpoints,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		left := items[i].UpdatedAt
		if left.IsZero() {
			left = items[i].CreatedAt
		}
		right := items[j].UpdatedAt
		if right.IsZero() {
			right = items[j].CreatedAt
		}
		if !left.Equal(right) {
			return left.After(right)
		}
		return items[i].ID < items[j].ID
	})
	return &ListResult{Items: items}, nil
}
