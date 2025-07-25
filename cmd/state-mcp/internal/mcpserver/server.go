package mcpserver

import (
	"context"
	"fmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ToolHandlerFunc func(context.Context, *primer.Values, mcp.CallToolRequest) (*mcp.CallToolResult, error)

// Handler wraps the MCP server and provides methods for adding tools and resources
type Handler struct {
	Server      *server.MCPServer
	primeGetter func() (*primer.Values, func() error, error)
}

func New(primeGetter func() (*primer.Values, func() error, error)) *Handler {
	s := server.NewMCPServer(
		constants.StateMCPCmd,
		constants.VersionNumber,
	)

	mcpHandler := &Handler{
		Server:      s,
		primeGetter: primeGetter,
	}

	return mcpHandler
}

func (m Handler) ServeStdio() error {
	if err := server.ServeStdio(m.Server); err != nil {
		logging.Error("Server error: %v\n", err)
	}
	return nil
}

// addResource adds a resource to the MCP server with error handling and logging
func (m *Handler) AddResource(resource mcp.Resource, handler server.ResourceHandlerFunc) {
	m.Server.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		r, err := handler(ctx, request)
		if err != nil {
			logging.Error("%s: Error handling resource request: %v", resource.Name, err)
			return nil, errs.Wrap(err, "Failed to handle resource request")
		}
		return r, nil
	})
}

// addPrompt adds a prompt to the MCP server with error handling and logging
func (m *Handler) AddPrompt(prompt mcp.Prompt, handler server.PromptHandlerFunc) {
	m.Server.AddPrompt(prompt, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		r, err := handler(ctx, request)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to handle prompt request")
		}
		return r, nil
	})
}

// addTool adds a tool to the MCP server with error handling and logging
func (m *Handler) AddTool(tool mcp.Tool, handler ToolHandlerFunc) {
	m.Server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (r *mcp.CallToolResult, rerr error) {
		p, closer, err := m.primeGetter()
		if err != nil {
			return nil, errs.Wrap(err, "Failed to get primer")
		}
		defer closer()
		r, err = handler(ctx, p, request)
		if err != nil {
			logging.Error("%s: Error handling tool request: %v", tool.Name, errs.JoinMessage(err))
			// Format all errors as a single string, so the client gets the full context
			return nil, fmt.Errorf("%s: %s", tool.Name, errs.JoinMessage(err))
		}
		return r, nil
	})
}
