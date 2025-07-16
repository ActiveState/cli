package toolregistry

import (
	"context"
	"strings"

	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/hello"
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