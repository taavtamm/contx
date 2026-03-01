# contx — Agent Guide

contx is a context-management tool for Claude Code. It stores reusable text snippets ("contexts") on disk and serves them to Claude Code as MCP resources that can be @-mentioned in conversations.

## What the tool does

Users create named context snippets (e.g. `company-info`, `api-docs`, `coding-style`). Each snippet has a name, optional description, optional tags, and a freeform markdown body. Once registered as an MCP server, Claude Code can reference any snippet with `@contx://global/<name>` or `@contx://project/<name>`.

## Architecture

```
main.go
cmd/              ← Cobra CLI commands
  root.go         ← entry point; adds all sub-commands
  serve.go        ← starts the MCP stdio server
  add.go          ← quick-add a context from stdin
  list.go         ← list contexts to stdout
  tui.go          ← launches the interactive TUI
  register.go     ← prints the `claude mcp add` command

internal/
  store/
    context.go    ← Context struct + marshal/unmarshal (YAML frontmatter + markdown body)
    store.go      ← Store interface + MultiStore (combines global + project stores)
    file_store.go ← FileStore: each context is one .md file in a directory
    discovery.go  ← FindProjectRoot, GlobalDir, ProjectDir helpers
  mcp/
    server.go     ← creates the mcp-go MCPServer
    resources.go  ← registers one MCP resource per context
    prompts.go    ← registers inject-context and list-contexts MCP prompts
  tui/            ← Bubble Tea TUI (list, form, confirm views, theming)
```

## Storage layout

Contexts are plain markdown files with YAML frontmatter, stored in one of two places:

- **Global** — `~/.contx/contexts/<name>.md` (shared across all projects)
- **Project** — `<project-root>/.contx/contexts/<name>.md`

Project root detection walks up the directory tree looking for `.contx/`, `.git`, `go.mod`, `package.json`, `Cargo.toml`, or `pyproject.toml`. The home directory is excluded from the `.contx` check so that `~/.contx` is never treated as a project root.

A context file looks like:

```markdown
---
name: company-info
description: Acme Corp overview
tags: [company, onboarding]
created_at: 2025-01-15T10:00:00Z
updated_at: 2025-01-15T10:00:00Z
---

Acme Corp is a B2B SaaS company focused on ...
```

## MCP interface

When running as `contx serve`, the server exposes:

- **Resources** — one static resource per context, URI scheme `contx://<scope>/<name>`, MIME type `text/plain`. Delivered as markdown: a `# Name` heading, optional italic description, then the body.
- **Prompts** — `inject-context` (takes a `uri` argument, injects that context as a user message) and `list-contexts` (returns a formatted list of all available contexts grouped by scope).

Register with Claude Code:

```sh
claude mcp add contx $(which contx) serve
# or user-scoped (all projects):
claude mcp add --scope user contx $(which contx) serve
```

## CLI commands

| Command | Description |
|---|---|
| `contx` | Launch the interactive TUI |
| `contx tui` | Same as above (explicit) |
| `contx serve` | Start the MCP stdio server |
| `contx add <name>` | Quick-add a context; body read from stdin |
| `contx list` | Print all contexts to stdout |
| `contx register` | Print the `claude mcp add` registration command |

`contx add` flags: `--global` (save to global scope), `--desc <string>`, `--tags <comma-separated>`.

Global flag on all commands: `--theme <name>` (choices: `tokyo-night`, `catppuccin`, `dracula`, `gruvbox`). Theme can also be set in `~/.contx/config.yaml`:

```yaml
theme: tokyo-night
```

## Key types

**`store.Context`** — the core data type. Fields: `Name`, `Description`, `Tags`, `CreatedAt`, `UpdatedAt`, `Scope` (runtime), `FilePath` (runtime), `Body` (runtime). URI is `contx://<scope>/<name>`.

**`store.Store`** (interface) — `List`, `Get`, `Save`, `Delete`, `Scope`. Implemented by `FileStore`.

**`store.MultiStore`** — combines a global `Store` and an optional project `Store`. `List` returns global first, then project. `Get` and `Delete` route by scope. `Save` routes by `c.Scope`; returns `ErrNoProjectRoot` if project scope is requested but no root was found.

## Development

```sh
make build    # produces bin/contx
make test     # runs go test ./...
make install  # installs to $GOPATH/bin
```

Dependencies are managed with Go modules (`go.mod`). Key external packages: `github.com/spf13/cobra`, `github.com/mark3labs/mcp-go`, `github.com/charmbracelet/bubbletea` (TUI), `gopkg.in/yaml.v3`.

## Conventions

- Context names must not contain spaces or slashes.
- Names are used directly as filenames (`<name>.md`), so keep them lowercase and hyphen-separated.
- The MCP server is statically configured at startup; adding a new context requires restarting the server for it to appear as a resource.
- Malformed `.md` files (missing or invalid frontmatter) are silently skipped during `List`.
- All timestamps are stored in UTC.
