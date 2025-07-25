package registry

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func ProjectPrompt() Prompt {
	return Prompt{
		Category: CategoryDebug,
		Prompt: mcp.NewPrompt("project",
			mcp.WithPromptDescription("A prompt to debug a project build failures"),
			mcp.WithArgument("prompt",
				mcp.ArgumentDescription("the user prompt with project information"),
			),
		),
		Handler: func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			prompt := request.Params.Arguments["prompt"]

			return mcp.NewGetPromptResult(
				"A prompt to debug a project build failures",
				[]mcp.PromptMessage{
					mcp.NewPromptMessage(
						mcp.RoleAssistant,
						mcp.NewTextContent(`
							You are BE-Copilot, a copilot for build engineers. You help build engineers with
							their task: adding missing dependencies to package builds.

							Projects are specified as <organization>/<project>.

							To identify errors in a project, you will:
							1. List all the failed ingredient builds for the project using list_project_build_failures
							2. Categorize the errors into three categories:
								- Dependencies error: The build failed because of missing or incorrect dependencies
								- Fixed: The build failed, but a newer revision of the ingredient exists that has
								  fixed the issue
								- Other errors: The build failed for reasons other than missing or incorrect dependencies
							3. Summarize this information without further debug or root cause analysis
							4. When asked about a specific ingredient version, you will download its logs using
							   download_logs andprovide its root cause analysis`,
						),
					),
					mcp.NewPromptMessage(
						mcp.RoleUser,
						mcp.NewTextContent(prompt),
					),
				},
			), nil
		},
	}
}

func IngredientPrompt() Prompt {
	return Prompt{
		Category: CategoryDebug,
		Prompt: mcp.NewPrompt("ingredient",
			mcp.WithPromptDescription("A prompt to debug a ingredient build failure"),
			mcp.WithArgument("prompt",
				mcp.ArgumentDescription("the user prompt with ingredient information"),
			),
		),
		Handler: func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			prompt := request.Params.Arguments["prompt"]

			return mcp.NewGetPromptResult(
				"A prompt to debug a project build failures",
				[]mcp.PromptMessage{
					mcp.NewPromptMessage(
						mcp.RoleAssistant,
						mcp.NewTextContent(`
							You are BE-Copilot, a copilot for build engineers. You help build engineers with
							their task: identify dependencies for package builds.

							Package and ingredients can be used interchangeably, as they refer to the same thing.

							When asked to fix or analyze a specific ingredient version, you will:
							1. Retrieve the ingredient details using get_ingredient_details
							2. List all files in the source code of the package version using list_source_code_files
							3. For each relevant file with dependency information, download its contents using
							   download_source_code_file
							4. Propose a fix honoring get_fix_instructions. The fix considers ALL dependencies that you
							   could retrieve by analyzing the source code, including but not limited to the build
							   backend (which must specifically define gte 0 as requirement)`,
						),
					),
					mcp.NewPromptMessage(
						mcp.RoleUser,
						mcp.NewTextContent(prompt),
					),
				},
			), nil
		},
	}
}
