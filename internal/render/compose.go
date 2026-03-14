package render

import (
	"context"
	"fmt"
	"strings"

	"github.com/xuenqlve/zygarde/internal/model"
)

// ComposeRenderer generates a minimal docker-compose artifact for the create flow.
type ComposeRenderer struct{}

// NewComposeRenderer creates a compose renderer instance.
func NewComposeRenderer() ComposeRenderer {
	return ComposeRenderer{}
}

// Render builds a minimal compose document from normalized runtime contexts.
func (ComposeRenderer) Render(_ context.Context, req Request) (*model.RenderResult, error) {
	var builder strings.Builder
	builder.WriteString("services:\n")

	services := make([]string, 0, len(req.Contexts))
	for _, item := range req.Contexts {
		serviceName := item.ServiceName
		if serviceName == "" {
			return nil, fmt.Errorf("render compose: service name is required")
		}
		services = append(services, serviceName)
		builder.WriteString(fmt.Sprintf("  %s:\n", serviceName))
		builder.WriteString("    image: alpine:3.20\n")
		builder.WriteString("    command: [\"sleep\", \"infinity\"]\n")
	}

	return &model.RenderResult{
		Content:        builder.String(),
		PrimaryFile:    req.Prepared.Layout.ComposeFile,
		ComposeVersion: "v2",
		Services:       services,
	}, nil
}
