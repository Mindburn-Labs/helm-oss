package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Catalog manages the registry of approved tools.
type Catalog interface {
	Search(ctx context.Context, query string) ([]ToolRef, error)
	Register(ctx context.Context, ref ToolRef) error
}

// ToolRef represents a tool reference for catalog search and definition.
type ToolRef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ServerID    string `json:"server_id,omitempty"`
	Schema      any    `json:"schema,omitempty"` // JSON schema (map[string]any or string)
}

// Validate checks that a ToolRef has a non-empty Name.
func (r ToolRef) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("tool ref name is required")
	}
	return nil
}

// ToolCatalog checks compliance and stores tool definitions.
type ToolCatalog struct {
	mu    sync.RWMutex
	tools map[string]ToolRef
}

func NewToolCatalog() *ToolCatalog {
	return &ToolCatalog{
		tools: make(map[string]ToolRef),
	}
}

// NewInMemoryCatalog is a constructor alias for tests
func NewInMemoryCatalog() *ToolCatalog {
	return NewToolCatalog()
}

func (c *ToolCatalog) RegisterCommonTools() {
	tools := []ToolRef{
		{
			Name:        "file_read",
			Description: "Read a UTF-8 text file from disk",
			ServerID:    "helm-governance",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string"},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "file_write",
			Description: "Write UTF-8 text content to disk",
			ServerID:    "helm-governance",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]any{"type": "string"},
					"content": map[string]any{"type": "string"},
				},
				"required": []string{"path", "content"},
			},
		},
	}

	for _, ref := range tools {
		_ = c.Register(context.Background(), ref)
	}
}

func (c *ToolCatalog) Register(ctx context.Context, ref ToolRef) error {
	if err := ref.Validate(); err != nil {
		return fmt.Errorf("invalid tool ref: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tools[ref.Name] = ref
	return nil
}

func (c *ToolCatalog) Search(ctx context.Context, query string) ([]ToolRef, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var results []ToolRef
	query = strings.ToLower(query)
	for _, tool := range c.tools {
		if strings.Contains(strings.ToLower(tool.Name), query) || strings.Contains(strings.ToLower(tool.Description), query) {
			results = append(results, tool)
		}
	}
	return results, nil
}

func (c *ToolCatalog) Lookup(name string) (ToolRef, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ref, ok := c.tools[name]
	return ref, ok
}

// ToolCallReceipt tracks the execution result (for Gap 10 audit).
type ToolCallReceipt struct {
	ID        string    `json:"id"`
	ToolName  string    `json:"tool_name"`
	Inputs    string    `json:"inputs"`
	Outputs   string    `json:"outputs"`
	Metadata  string    `json:"metadata"`
	Timestamp time.Time `json:"timestamp"`
}

func (c *ToolCatalog) AuditToolCall(name string, params map[string]any, result any) (ToolCallReceipt, error) {
	inputJSON, err := json.Marshal(params)
	if err != nil {
		return ToolCallReceipt{}, fmt.Errorf("failed to marshal tool call inputs: %w", err)
	}
	outputJSON, err := json.Marshal(result)
	if err != nil {
		return ToolCallReceipt{}, fmt.Errorf("failed to marshal tool call outputs: %w", err)
	}

	return ToolCallReceipt{
		ID:        fmt.Sprintf("call-%d", time.Now().UnixNano()),
		ToolName:  name,
		Inputs:    string(inputJSON),
		Outputs:   string(outputJSON),
		Timestamp: time.Now(),
	}, nil
}
