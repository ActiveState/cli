package registry

import (
	"context"
	"slices"

	"github.com/ActiveState/cli/internal/primer"
	"github.com/mark3labs/mcp-go/mcp"
)

type Tool struct {
	mcp.Tool
	Category Category
	Handler  func(context.Context, *primer.Values, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

type Prompt struct {
	mcp.Prompt
	Category Category
	Handler  func(context.Context, mcp.GetPromptRequest) (*mcp.GetPromptResult, error)
}

type Registry struct {
	tools   map[Category][]Tool
	prompts map[Category][]Prompt
}

func New() *Registry {
	r := &Registry{
		tools:   make(map[Category][]Tool),
		prompts: make(map[Category][]Prompt),
	}

	r.RegisterTool(ProjectErrorsTool())
	r.RegisterTool(DownloadLogsTool())
	r.RegisterTool(GetInstructionsTool())
	r.RegisterTool(ListSourceFilesTool())
	r.RegisterTool(DownloadSourceFileTool())
	r.RegisterTool(GetIngredientDetailsTool())

	r.RegisterPrompt(ProjectPrompt())
	r.RegisterPrompt(IngredientPrompt())

	return r
}

func (r *Registry) RegisterTool(tool Tool) {
	if _, ok := r.tools[tool.Category]; !ok {
		r.tools[tool.Category] = []Tool{}
	}
	r.tools[tool.Category] = append(r.tools[tool.Category], tool)
}

func (r *Registry) RegisterPrompt(prompt Prompt) {
	if _, ok := r.prompts[prompt.Category]; !ok {
		r.prompts[prompt.Category] = []Prompt{}
	}
	r.prompts[prompt.Category] = append(r.prompts[prompt.Category], prompt)
}

func (r *Registry) GetTools(requestCategories ...string) []Tool {
	if len(requestCategories) == 0 {
		for _, category := range GetCategories() {
			if category == CategoryDebug {
				// Debug must be explicitly requested
				continue
			}
			requestCategories = append(requestCategories, string(category))
		}
	}
	categories := GetCategories()
	result := []Tool{}
	for _, category := range categories {
		if slices.Contains(requestCategories, string(category)) {
			result = append(result, r.tools[category]...)
		}
	}
	return result
}

func (r *Registry) GetPrompts(requestCategories ...string) []Prompt {
	if len(requestCategories) == 0 {
		for _, category := range GetCategories() {
			if category == CategoryDebug {
				// Debug must be explicitly requested
				continue
			}
			requestCategories = append(requestCategories, string(category))
		}
	}
	categories := GetCategories()
	result := []Prompt{}
	for _, category := range categories {
		if slices.Contains(requestCategories, string(category)) {
			result = append(result, r.prompts[category]...)
		}
	}
	return result
}
