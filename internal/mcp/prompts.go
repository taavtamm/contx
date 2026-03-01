package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/taavtamm/contx/internal/store"
)

// RegisterPrompts registers the inject-context and list-contexts MCP prompts.
func RegisterPrompts(s *server.MCPServer, ms *store.MultiStore) {
	// inject-context: given a URI, return the context body as a prompt message
	s.AddPrompt(mcp.NewPrompt(
		"inject-context",
		mcp.WithPromptDescription("Inject a contx context into the conversation"),
		mcp.WithArgument("uri",
			mcp.ArgumentDescription("The context URI, e.g. contx://global/company-info"),
			mcp.RequiredArgument(),
		),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return handleInjectContext(ms, req)
	})

	// list-contexts: returns a summary of all available contexts
	s.AddPrompt(mcp.NewPrompt(
		"list-contexts",
		mcp.WithPromptDescription("List all available contx contexts"),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return handleListContexts(ms)
	})

	// global: inject all global contexts
	s.AddPrompt(mcp.NewPrompt(
		"global",
		mcp.WithPromptDescription("Inject all global contx contexts into the conversation"),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return handleInjectScopedContexts(ms, store.ScopeGlobal)
	})

	// project: inject all project-scoped contexts
	s.AddPrompt(mcp.NewPrompt(
		"project",
		mcp.WithPromptDescription("Inject all project contx contexts into the conversation"),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return handleInjectScopedContexts(ms, store.ScopeProject)
	})
}

func handleInjectContext(ms *store.MultiStore, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	uri := req.Params.Arguments["uri"]
	if uri == "" {
		return nil, fmt.Errorf("uri argument is required")
	}

	scope, name, err := parseURI(uri)
	if err != nil {
		return nil, err
	}

	c, err := ms.Get(scope, name)
	if err != nil {
		return nil, fmt.Errorf("context not found: %s", uri)
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Context: %s", c.Name),
		Messages: []mcp.PromptMessage{
			{
				Role:    mcp.RoleUser,
				Content: mcp.NewTextContent(formatContext(c)),
			},
		},
	}, nil
}

func handleListContexts(ms *store.MultiStore) (*mcp.GetPromptResult, error) {
	contexts, err := ms.List()
	if err != nil {
		return nil, fmt.Errorf("list contexts: %w", err)
	}

	if len(contexts) == 0 {
		return &mcp.GetPromptResult{
			Description: "No contexts found",
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleUser,
					Content: mcp.NewTextContent("No contx contexts are defined yet. Run `contx` to open the TUI and create some."),
				},
			},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString("# Available contx contexts\n\n")

	var global, project []*store.Context
	for _, c := range contexts {
		if c.Scope == store.ScopeGlobal {
			global = append(global, c)
		} else {
			project = append(project, c)
		}
	}

	if len(global) > 0 {
		sb.WriteString("## Global\n\n")
		for _, c := range global {
			sb.WriteString(fmt.Sprintf("- **%s** — %s\n  URI: `%s`\n", c.Name, c.Description, c.URI()))
		}
		sb.WriteString("\n")
	}
	if len(project) > 0 {
		sb.WriteString("## Project\n\n")
		for _, c := range project {
			sb.WriteString(fmt.Sprintf("- **%s** — %s\n  URI: `%s`\n", c.Name, c.Description, c.URI()))
		}
	}

	return &mcp.GetPromptResult{
		Description: "Available contx contexts",
		Messages: []mcp.PromptMessage{
			{
				Role:    mcp.RoleUser,
				Content: mcp.NewTextContent(sb.String()),
			},
		},
	}, nil
}

func handleInjectScopedContexts(ms *store.MultiStore, scope store.Scope) (*mcp.GetPromptResult, error) {
	all, err := ms.List()
	if err != nil {
		return nil, fmt.Errorf("list contexts: %w", err)
	}

	var filtered []*store.Context
	for _, c := range all {
		if c.Scope == scope {
			filtered = append(filtered, c)
		}
	}

	if len(filtered) == 0 {
		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("No %s contexts found", scope),
			Messages: []mcp.PromptMessage{
				{
					Role:    mcp.RoleUser,
					Content: mcp.NewTextContent(fmt.Sprintf("No %s contx contexts are defined yet.", scope)),
				},
			},
		}, nil
	}

	var sb strings.Builder
	for i, c := range filtered {
		if i > 0 {
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString(formatContext(c))
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("All %s contexts (%d)", scope, len(filtered)),
		Messages: []mcp.PromptMessage{
			{
				Role:    mcp.RoleUser,
				Content: mcp.NewTextContent(sb.String()),
			},
		},
	}, nil
}

// parseURI parses "contx://global/name" or "contx://project/name".
func parseURI(uri string) (store.Scope, string, error) {
	const prefix = "contx://"
	if !strings.HasPrefix(uri, prefix) {
		return "", "", fmt.Errorf("invalid URI scheme: %s", uri)
	}
	rest := uri[len(prefix):]
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[1] == "" {
		return "", "", fmt.Errorf("invalid URI format: %s", uri)
	}
	scope := store.Scope(parts[0])
	if scope != store.ScopeGlobal && scope != store.ScopeProject {
		return "", "", fmt.Errorf("unknown scope %q in URI: %s", parts[0], uri)
	}
	return scope, parts[1], nil
}
