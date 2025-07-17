package toolregistry

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/graphql"
	"github.com/ActiveState/cli/internal/runners/hello"
	"github.com/ActiveState/cli/pkg/platform/api"
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

type BuildNodesResponse struct {
	Commit struct {
		Build struct {
			Nodes []BuildNode `json:"nodes"`
		} `json:"build"`
	} `json:"commit"`
}

type BuildNode struct {
	Typename            string `json:"__typename"`
	Name                string `json:"name"`
	Namespace           string `json:"namespace"`
	Version             string `json:"version"`
	DisplayName         string `json:"displayName"`
	LogURL              string `json:"logURL"`
	Status              string `json:"status"`
	LastBuildFinishedAt string `json:"lastBuildFinishedAt"`
}

func ProjectErrorsTool() Tool {
	return Tool{
		Category: ToolCategoryDebug,
		Tool: mcp.NewTool(
			"projecterrors",
			mcp.WithDescription("Project errors tool - shows errors for a project"),
			mcp.WithString("namespace", mcp.Description("Project namespace in format 'owner/project'")),
		),
		Handler: func(ctx context.Context, p *primer.Values, mcpRequest mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			namespace, _ := mcpRequest.RequireString("namespace")

			ns, err := project.ParseNamespace(namespace)
			if err != nil {
				return mcp.NewToolResultError("Invalid namespace format. Use 'owner/project' format."), nil
			}

			runner := graphql.New(p, api.ServiceBuildPlanner)
			request := &graphql.Request{
				QueryStr: `query($organization: String!, $project: String!) {
								project(organization: $organization, project: $project) {
									... on Project {
										commit {
											... on Commit {
												build {
													... on Build {
														nodes {
															... on Source {
																__typename, name, version, namespace
															}
															... on ArtifactPermanentlyFailed {
																__typename, displayName, logURL, status, lastBuildFinishedAt
															}
														}
													}
												}
											}
										}
									}
								}
							}`,
				QueryVars: map[string]interface{}{
					"organization": ns.Owner,
					"project":      ns.Project,
				},
			}
			response := BuildNodesResponse{}
			err = runner.Run(request, &response)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			// at this point we can handle request to create the final output
			// OR just return the full response as text
			jsonBytes, _ := json.Marshal(response)
			p.Output().Print(string(jsonBytes))

			return mcp.NewToolResultText(
				strings.Join(p.Output().History().Print, "\n"),
			), nil
		},
	}
}
