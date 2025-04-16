package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ActiveState/cli/cmd/state/donotshipme"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/scriptrun"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/mark3labs/mcp-go/mcp"
)

func registerScriptTools(mcpHandler *mcpServerHandler) func() error {
	byt := &bytes.Buffer{}
	prime, close, err := mcpHandler.newPrimer(os.Getenv(constants.ActivatedStateEnvVarName), byt)
	if err != nil {
		panic(err)
	}

	scripts, err := prime.Project().Scripts()
	if err != nil {
		panic(err)
	}

	for _, script := range scripts {
		mcpHandler.addTool(mcp.NewTool(script.Name(),
			mcp.WithDescription(script.Description()),
		), func(ctx context.Context, request mcp.CallToolRequest) (r *mcp.CallToolResult, rerr error) {
			byt.Truncate(0)

			scriptrunner := scriptrun.New(prime)
			if !script.Standalone() && scriptrunner.NeedsActivation() {
				if err := scriptrunner.PrepareVirtualEnv(); err != nil {
					return nil, errs.Wrap(err, "Failed to prepare virtual environment")
				}
			}

			err := scriptrunner.Run(script, []string{})
			if err != nil {
				return nil, errs.Wrap(err, "Failed to run script")
			}

			return mcp.NewToolResultText(byt.String()), nil
		})
	}

	return close
}

// registerCuratedTools registers a curated set of tools for the AI assistant
func registerCuratedTools(mcpHandler *mcpServerHandler) {
	projectDirParam := mcp.WithString("project_directory",
		mcp.Required(),
		mcp.Description("Absolute path to the directory where your activestate project is checked out. It should contain the activestate.yaml file."),
	)

	mcpHandler.addTool(mcp.NewTool("list_projects",
		mcp.WithDescription("List all ActiveState projects checked out on the local machine"),
	), mcpHandler.listProjectsHandler)

	mcpHandler.addTool(mcp.NewTool("view_manifest",
		mcp.WithDescription("Show the manifest (packages and dependencies) for a locally checked out ActiveState platform project"),
		projectDirParam,
	), mcpHandler.manifestHandler)

	mcpHandler.addTool(mcp.NewTool("view_cves",
		mcp.WithDescription("Show the CVEs for a locally checked out ActiveState platform project"),
		projectDirParam,
	), mcpHandler.cveHandler)

	mcpHandler.addTool(mcp.NewTool("lookup_cve",
		mcp.WithDescription("Lookup one or more CVEs by their ID"),
		mcp.WithString("cve_ids",
			mcp.Required(),
			mcp.Description("The IDs of the CVEs to lookup, comma separated"),
		),
	), mcpHandler.lookupCveHandler)
}

// registerRawTools registers all State Tool commands as raw tools
func registerRawTools(mcpHandler *mcpServerHandler) func() error {
	byt := &bytes.Buffer{}
	prime, close, err := mcpHandler.newPrimer("", byt)
	if err != nil {
		panic(err)
	}

	require := func(b bool) mcp.PropertyOption {
		if b {
			return mcp.Required()
		}
		return func(map[string]interface{}) {}
	}

	tree := donotshipme.CmdTree(prime)
	for _, command := range tree.Command().AllChildren() {
		// Best effort to filter out interactive commands
		if sliceutils.Contains([]string{"activate", "shell"}, command.NameRecursive()) {
			continue
		}

		opts := []mcp.ToolOption{
			mcp.WithDescription(command.Description()),
		}

		// Require project directory for most commands. This is currently not encoded into the command tree
		if !sliceutils.Contains([]string{"projects", "auth"}, command.BaseCommand().Name()) {
			opts = append(opts, mcp.WithString(
				"project_directory",
				mcp.Required(),
				mcp.Description("Absolute path to the directory where your activestate project is checked out. It should contain the activestate.yaml file."),
			))
		}

		for _, arg := range command.Arguments() {
			opts = append(opts, mcp.WithString(arg.Name,
				require(arg.Required),
				mcp.Description(arg.Description),
			))
		}
		for _, flag := range command.Flags() {
			opts = append(opts, mcp.WithString(flag.Name,
				mcp.Description(flag.Description),
			))
		}
		mcpHandler.addTool(
			mcp.NewTool(strings.Join(strings.Split(command.NameRecursive(), " "), "_"), opts...),
			func(ctx context.Context, request mcp.CallToolRequest) (r *mcp.CallToolResult, rerr error) {
				byt.Truncate(0)
				if projectDir, ok := request.Params.Arguments["project_directory"]; ok {
					pj, err := project.FromPath(projectDir.(string))
					if err != nil {
						return nil, errs.Wrap(err, "Failed to create project")
					}
					prime.SetProject(pj)
				}
				// Reinitialize tree with updated primer, because currently our command can take things
				// from the primer at the time of registration, and not the time of invocation.
				invocationTree := donotshipme.CmdTree(prime)
				for _, child := range invocationTree.Command().AllChildren() {
					if child.NameRecursive() == command.NameRecursive() {
						command = child
						break
					}
				}
				args := strings.Split(command.NameRecursive(), " ")
				for _, arg := range command.Arguments() {
					v, ok := request.Params.Arguments[arg.Name]
					if !ok {
						break
					}
					args = append(args, v.(string))
				}
				for _, flag := range command.Flags() {
					v, ok := request.Params.Arguments[flag.Name]
					if !ok {
						break
					}
					args = append(args, fmt.Sprintf("--%s=%s", flag.Name, v.(string)))
				}
				logging.Debug("Executing command: %s, args: %v (%v)", command.NameRecursive(), args, args == nil)
				err := command.Execute(args)
				if err != nil {
					return nil, errs.Wrap(err, "Failed to execute command")
				}
				return mcp.NewToolResultText(byt.String()), nil
			},
		)
	}

	return close
} 