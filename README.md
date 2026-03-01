# contx

Context management for Claude Code. Store reusable text snippets and serve them as MCP resources.

## Install

```sh
go install github.com/taavtamm/contx@latest
```

## Register with Claude Code

```sh
# Project-scoped
claude mcp add contx $(which contx) serve

# User-scoped (all projects)
claude mcp add --scope user contx $(which contx) serve
```

## Usage

```sh
contx              # open TUI
contx serve        # start MCP server
contx add <name>   # add context from stdin
contx list         # list all contexts
```

In Claude Code, use `@contx global` or `@contx project` to inject all contexts of that scope, or reference a specific one with `@contx inject-context` and its URI.

## Storage

- **Global** — `~/.contx/contexts/<name>.md`
- **Project** — `<project-root>/.contx/contexts/<name>.md`

## Themes

`tokyo-night` (default), `catppuccin`, `dracula`, `gruvbox`

```sh
contx --theme dracula
# or set in ~/.contx/config.yaml:
# theme: dracula
```
