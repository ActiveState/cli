package toolregistry

import (
	"context"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/hello"
	"github.com/ActiveState/cli/internal/runners/mcp/downloadlogs"
	"github.com/ActiveState/cli/internal/runners/mcp/downloadsource"
	"github.com/ActiveState/cli/internal/runners/mcp/ingredientdetails"
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
			mcp.WithDescription("Retrieves all the failed artifact builds for a specific project"),
			mcp.WithString("namespace", mcp.Description("Project namespace in format 'owner/project'")),
		),
		Handler: func(ctx context.Context, p *primer.Values, mcpRequest mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			namespace, err := mcpRequest.RequireString("namespace")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("a project in the format 'owner/project' is required: %s", errs.JoinMessage(err))), nil
			}

			ns, err := project.ParseNamespace(namespace)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error parsing project namespace: %s", errs.JoinMessage(err))), nil
			}

			runner := projecterrors.New(p, ns)
			err = runner.Run()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error executing GraphQL query: %s", errs.JoinMessage(err))), nil
			}

			return mcp.NewToolResultText(
				strings.Join(p.Output().History().Print, "\n"),
			), nil
		},
	}
}

func DownloadLogsTool() Tool {
	return Tool{
		Category: ToolCategoryDebug,
		Tool: mcp.NewTool(
			"download_logs",
			mcp.WithDescription("Downloads logs from a specified URL"),
			mcp.WithString("url", mcp.Description("The URL to download logs from"), mcp.Required()),
		),
		Handler: func(ctx context.Context, p *primer.Values, mcpRequest mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			url, err := mcpRequest.RequireString("url")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("a URL is required: %s", errs.JoinMessage(err))), nil
			}

			runner := downloadlogs.New(p, url)
			err = runner.Run()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error downloading logs: %s", errs.JoinMessage(err))), nil
			}

			return mcp.NewToolResultText(
				strings.Join(p.Output().History().Print, "\n"),
			), nil
		},
	}
}

func GetIngredientDetailsTool() Tool {
	return Tool{
		Category: ToolCategoryDebug,
		Tool: mcp.NewTool(
			"get_ingredient_details",
			mcp.WithDescription("Retrieves the details for a specified ingredient, including its dependencies, status, and source URI"),
			mcp.WithString("package", mcp.Description("The package to retrieve the source URI for"), mcp.Required()),
			mcp.WithString("version", mcp.Description("The version of the package to retrieve the source URI for"), mcp.Required()),
			mcp.WithString("namespace", mcp.Description("The language namespace of the package to retrieve the source URI for (e.g. language/python)"), mcp.Required()),
		),
		Handler: func(ctx context.Context, p *primer.Values, mcpRequest mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name, err := mcpRequest.RequireString("package")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("a package name is required: %s", errs.JoinMessage(err))), nil
			}

			version, err := mcpRequest.RequireString("version")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("a package version is required: %s", errs.JoinMessage(err))), nil
			}

			namespace, err := mcpRequest.RequireString("namespace")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("a namespace is required: %s", errs.JoinMessage(err))), nil
			}

			runner := ingredientdetails.New(p, name, version, namespace)
			err = runner.Run()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error executing GraphQL query: %s", errs.JoinMessage(err))), nil
			}

			return mcp.NewToolResultText(
				strings.Join(p.Output().History().Print, "\n"),
			), nil
		},
	}
}

func ListSourceFilesTool() Tool {
	return Tool{
		Category: ToolCategoryDebug,
		Tool: mcp.NewTool(
			"list_source_code_files",
			mcp.WithDescription("Lists source code files from a specified source URI (HTTPS or S3)"),
			mcp.WithString("sourceUri", mcp.Description("The URI (e.g. .tar.gz) to list source code files from"), mcp.Required()),
		),
		Handler: func(ctx context.Context, p *primer.Values, mcpRequest mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sourceUri, err := mcpRequest.RequireString("sourceUri")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("a source URI is required: %s", errs.JoinMessage(err))), nil
			}

			runner := downloadsource.New(p, sourceUri, "")
			err = runner.Run()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error downloading logs: %s", errs.JoinMessage(err))), nil
			}

			return mcp.NewToolResultText(
				strings.Join(p.Output().History().Print, "\n"),
			), nil
		},
	}
}

func DownloadSourceFileTool() Tool {
	return Tool{
		Category: ToolCategoryDebug,
		Tool: mcp.NewTool(
			"download_source_code_file",
			mcp.WithDescription("Downloads a specific source code file from a specified archive URI"),
			mcp.WithString("sourceUri", mcp.Description("The archive URI (e.g. .tar.gz) to download the source code file from"), mcp.Required()),
			mcp.WithString("targetFile", mcp.Description("The target source code file to download"), mcp.Required()),
		),
		Handler: func(ctx context.Context, p *primer.Values, mcpRequest mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sourceUri, err := mcpRequest.RequireString("sourceUri")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("a source URI is required: %s", errs.JoinMessage(err))), nil
			}

			targetFile, err := mcpRequest.RequireString("targetFile")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("a target file is required: %s", errs.JoinMessage(err))), nil
			}

			runner := downloadsource.New(p, sourceUri, targetFile)
			err = runner.Run()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error downloading source file: %s", errs.JoinMessage(err))), nil
			}

			return mcp.NewToolResultText(
				strings.Join(p.Output().History().Print, "\n"),
			), nil
		},
	}
}

func GetInstructionsTool() Tool {
	return Tool{
		Category: ToolCategoryDebug,
		Tool: mcp.NewTool(
			"get_fix_instructions",
			mcp.WithDescription("Retrieves the fix format and instructions in which fixes must be provided"),
		),
		Handler: func(ctx context.Context, p *primer.Values, mcpRequest mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			instructions := map[string]string{
				"format": `
					- description: Extracted from source distribution files X, Y, and Z.
					feature: python-dotenv
					namespace: language/python
					original_requirement: python-dotenv >=0.21.0
					conditions:
						- feature: python
						namespace: language
						requirements:
							- comparator: eq
							version: 3.10
					requirements:
						- comparator: gte
						version: 0.21.0
					type: runtime
				`,
				"instructions": `
					For dependencies needed in both build and runtime phases, create separate dependency lines (one with type: build, one with type: runtime).
					If an individual file is missing, such as README.md or versions.txt, the issue might be that the source code used for build did not include these files by mistake.
				`,
			}
			return mcp.NewToolResultText(fmt.Sprintf("%v", instructions)), nil
		},
	}
}
