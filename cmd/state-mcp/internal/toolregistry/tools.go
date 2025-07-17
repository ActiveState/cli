package toolregistry

import (
	"context"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/hello"
	"github.com/ActiveState/cli/internal/runners/mcp/projecterrors"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/mark3labs/mcp-go/mcp"
)

func HelloWorldTool() Tool {
	return Tool{
		Category: ToolCategoryDebug,
		Tool: mcp.NewTool(
			"hello",
			mcp.WithDescription("Hello world tool"),
			mcp.WithString("name", mcp.Required(), mcp.Description("The name to say hello to")),
		),
		Handler: func(ctx context.Context, p *primer.Values, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name, err := request.RequireString("name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			runner := hello.New(p)
			params := hello.NewParams()
			params.Name = name

			err = runner.Run(params)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(
				strings.Join(p.Output().History().Print, "\n"),
			), nil
		},
	}
}

func ProjectErrorsTool() Tool {
	return Tool{
		Category: ToolCategoryDebug,
		Tool: mcp.NewTool(
			"list_project_build_failures",
			mcp.WithDescription("Retrieves all the failed builds for a specific project"),
			mcp.WithString("namespace", mcp.Description("Project namespace in format 'owner/project'")),
		),
		Handler: func(ctx context.Context, p *primer.Values, mcpRequest mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			namespace, _ := mcpRequest.RequireString("namespace")

			ns, err := project.ParseNamespace(namespace)
			if err != nil {
				return mcp.NewToolResultError(fmt.Errorf("invalid namespace format. Use 'owner/project' format: %w", err).Error()), nil
			}

			runner := projecterrors.New(p, ns)
			err = runner.Run()
			if err != nil {
				return mcp.NewToolResultError(fmt.Errorf("error executing GraphQL query: %v", err).Error()), nil
			}

			return mcp.NewToolResultText(
				strings.Join(p.Output().History().Print, "\n"),
			), nil
		},
	}
}
