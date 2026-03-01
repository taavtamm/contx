package mcp

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/taavtamm/contx/internal/store"
)

// NewServer creates and configures the MCP stdio server.
func NewServer(ms *store.MultiStore) *server.MCPServer {
	s := server.NewMCPServer(
		"contx",
		"1.0.0",
		server.WithResourceCapabilities(true, false),
		server.WithPromptCapabilities(false),
	)

	RegisterResources(s, ms)
	RegisterPrompts(s, ms)

	return s
}
