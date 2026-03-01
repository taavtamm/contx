package store

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Scope string

const (
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"
)

type Context struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Tags        []string  `yaml:"tags,omitempty"`
	CreatedAt   time.Time `yaml:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at"`

	// Runtime-only fields (not stored in frontmatter)
	Scope    Scope  `yaml:"-"`
	FilePath string `yaml:"-"`
	Body     string `yaml:"-"`
}

func (c *Context) URI() string {
	return "contx://" + string(c.Scope) + "/" + c.Name
}

// Marshal serializes the Context to markdown with YAML frontmatter.
func (c *Context) Marshal() ([]byte, error) {
	fm, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fm)
	buf.WriteString("---\n")
	if c.Body != "" {
		buf.WriteString("\n")
		buf.WriteString(c.Body)
		if !strings.HasSuffix(c.Body, "\n") {
			buf.WriteString("\n")
		}
	}
	return buf.Bytes(), nil
}

// Unmarshal parses markdown with YAML frontmatter into a Context.
func Unmarshal(data []byte) (*Context, error) {
	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		return nil, fmt.Errorf("missing frontmatter delimiter")
	}

	rest := content[4:]
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return nil, fmt.Errorf("missing closing frontmatter delimiter")
	}

	fmRaw := rest[:end]
	body := rest[end+5:] // skip "\n---\n"

	// Trim leading newline from body if present
	body = strings.TrimPrefix(body, "\n")

	var c Context
	if err := yaml.Unmarshal([]byte(fmRaw), &c); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	c.Body = body
	return &c, nil
}
