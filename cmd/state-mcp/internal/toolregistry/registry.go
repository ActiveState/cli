package toolregistry

import (
	"context"
	"slices"

	"github.com/ActiveState/cli/internal/primer"
	"github.com/mark3labs/mcp-go/mcp"
)

type Tool struct {
	mcp.Tool
	Category ToolCategory
	Handler func(context.Context, *primer.Values, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

type Registry struct {
	tools map[ToolCategory][]Tool
}

func New() *Registry {
	r := &Registry{
		tools: make(map[ToolCategory][]Tool),
	}

	r.RegisterTool(HelloWorldTool())

	return r
}

func (r *Registry) RegisterTool(tool Tool) {
	if _, ok := r.tools[tool.Category]; !ok {
		r.tools[tool.Category] = []Tool{}
	}
	r.tools[tool.Category] = append(r.tools[tool.Category], tool)
}

func (r *Registry) GetTools(requestCategories ...string) []Tool {
	if len(requestCategories) == 0 {
		for _, category := range Categories() {
			if category == ToolCategoryDebug {
				// Debug must be explicitly requested
				continue
			}
			requestCategories = append(requestCategories, string(category))
		}
	}
	categories := Categories()
	result := []Tool{}
	for _, category := range categories {
		if slices.Contains(requestCategories, string(category)) {
			result = append(result, r.tools[category]...)
		}
	}
	return result
}