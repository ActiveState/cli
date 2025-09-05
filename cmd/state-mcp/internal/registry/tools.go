package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/hello"
	"github.com/ActiveState/cli/internal/runners/mcp/createrevision"
	"github.com/ActiveState/cli/internal/runners/mcp/downloadlogs"
	"github.com/ActiveState/cli/internal/runners/mcp/downloadsource"
	"github.com/ActiveState/cli/internal/runners/mcp/ingredientdetails"
	"github.com/ActiveState/cli/internal/runners/mcp/projecterrors"
	"github.com/ActiveState/cli/internal/runners/mcp/rebuildproject"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/mark3labs/mcp-go/mcp"
)

func HelloWorldTool() Tool {
	return Tool{
		Category: CategoryDebug,
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
		Category: CategoryDebug,
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

			runner := projecterrors.New(p)
			params := projecterrors.NewParams(ns)
			err = runner.Run(params)
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
		Category: CategoryDebug,
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

			runner := downloadlogs.New(p)
			params := downloadlogs.NewParams()
			params.LogUrl = url
			err = runner.Run(params)
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
		Category: CategoryDebug,
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

			runner := ingredientdetails.New(p)
			params := ingredientdetails.NewParams(name, version, namespace)
			err = runner.Run(params)
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
		Category: CategoryDebug,
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

			runner := downloadsource.New(p)
			params := downloadsource.NewParams(sourceUri, "")
			err = runner.Run(params)
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
		Category: CategoryDebug,
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

			runner := downloadsource.New(p)
			params := downloadsource.NewParams(sourceUri, targetFile)
			err = runner.Run(params)
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
		Category: CategoryDebug,
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

func CreateIngredientRevisionTool() Tool {
	return Tool{
		Category: CategoryDebug,
		Tool: mcp.NewTool(
			"create_ingredient_revision",
			mcp.WithDescription("Creates a new revision for the specified ingredient version"),
			mcp.WithString("namespace", mcp.Description("The namespace of the ingredient, e.g. language/python"), mcp.Required()),
			mcp.WithString("name", mcp.Description("The name of the ingredient, e.g. numpy"), mcp.Required()),
			mcp.WithString("version", mcp.Description("The version of the ingredient, e.g. 0.1.0"), mcp.Required()),
			mcp.WithString("dependencies", mcp.Description(`The JSON representation of dependencies, e.g.
				[ { "conditions": [ { "feature": "alternative-built-language", "namespace": "language", "requirements": [{"comparator": "eq", "sortable_version": []}] } ], "description": "Camel build dependency", "feature": "camel", "namespace": "builder", "requirements": [{"comparator": "gte", "sortable_version": ["0"], "version": "0"}], "type": "build" }, { "conditions": null, "description": "Extracted from source distribution in PyPI.", "feature": "cython", "namespace": "language/python", "original_requirement": "Cython <3.0,>=0.29.24", "requirements": [ {"comparator": "gte", "sortable_version": ["0","0","29","24"], "version": "0.29.24"}, {"comparator": "lt", "sortable_version": ["0","3"], "version": "3.0"} ], "type": "build" }, { "conditions": null, "description": "Extracted from source distribution in PyPI.", "feature": "setuptools", "namespace": "language/python", "original_requirement": "setuptools ==59.2.0", "requirements": [{"comparator": "eq", "sortable_version": ["0","59","2"], "version": "59.2.0"}], "type": "runtime" } ]
			`), mcp.Required()),
			mcp.WithString("comment", mcp.Description("A short summary of the changes you made, and why you made them - including the file that declares an added or updated dependency, e.g. updated dependencies and python version, as per pyproject.toml"), mcp.Required()),
		),
		Handler: func(ctx context.Context, p *primer.Values, mcpRequest mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			namespace, err := mcpRequest.RequireString("namespace")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("an ingredient namespace str is required: %s", errs.JoinMessage(err))), nil
			}
			name, err := mcpRequest.RequireString("name")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("an ingredient name str is required: %s", errs.JoinMessage(err))), nil
			}
			version, err := mcpRequest.RequireString("version")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("an ingredient version str is required: %s", errs.JoinMessage(err))), nil
			}
			dependencies, err := mcpRequest.RequireString("dependencies")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("the ingredient dependencies str is required: %s", errs.JoinMessage(err))), nil
			}
			comment, err := mcpRequest.RequireString("comment")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("a comment str for the ingredient is required: %s", errs.JoinMessage(err))), nil
			}

			params := createrevision.NewParams(namespace, name, version, dependencies, comment)

			runner := createrevision.New(p)

			err = runner.Run(params)

			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error creating new ingredient version revision: %s", errs.JoinMessage(err))), nil
			}

			return mcp.NewToolResultText(
				strings.Join(p.Output().History().Print, "\n"),
			), nil
		},
	}
}

func RebuildProjectTool() Tool {
	return Tool{
		Category: CategoryDebug,
		Tool: mcp.NewTool(
			"rebuild_project",
			mcp.WithDescription("Triggers a project rebuild after all errors have been addressed"),
			mcp.WithString("project", mcp.Description("Project namespace in format 'owner/project'"), mcp.Required()),
		),
		Handler: func(ctx context.Context, p *primer.Values, mcpRequest mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			namespace, err := mcpRequest.RequireString("project")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("a project in the format 'owner/project' is required: %s", errs.JoinMessage(err))), nil
			}
			ns, err := project.ParseNamespace(namespace)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error parsing project namespace: %s", errs.JoinMessage(err))), nil
			}

			params := rebuildproject.NewParams()
			params.Namespace = ns

			runner := rebuildproject.New(p)
			err = runner.Run(params)

			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("error rebuilding project: %s", errs.JoinMessage(err))), nil
			}

			return mcp.NewToolResultText(
				strings.Join(p.Output().History().Print, "\n"),
			), nil
		},
	}
}
