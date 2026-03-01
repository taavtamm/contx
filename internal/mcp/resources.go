package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/taavtamm/contx/internal/store"
)

// RegisterResources registers one static MCP resource per context in the multi-store.
func RegisterResources(s *server.MCPServer, ms *store.MultiStore) {
	contexts, err := ms.List()
	if err != nil {
		return
	}

	for _, c := range contexts {
		resource := mcp.NewResource(
			c.URI(),
			c.Name,
			mcp.WithResourceDescription(c.Description),
			mcp.WithMIMEType("text/plain"),
		)

		s.AddResource(resource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return handleReadResource(ms, c.Scope, c.Name)
		})
	}
}

func handleReadResource(ms *store.MultiStore, scope store.Scope, name string) ([]mcp.ResourceContents, error) {
	c, err := ms.Get(scope, name)
	if err != nil {
		return nil, fmt.Errorf("context %s/%s not found: %w", scope, name, err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      c.URI(),
			MIMEType: "text/plain",
			Text:     formatContext(c),
		},
	}, nil
}

// formatContext renders a context as a markdown string for MCP delivery.
func formatContext(c *store.Context) string {
	s := fmt.Sprintf("# %s\n\n", c.Name)
	if c.Description != "" {
		s += fmt.Sprintf("*%s*\n\n", c.Description)
	}
	return s + c.Body
}
