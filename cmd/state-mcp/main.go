package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/cmd/state/donotshipme"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/cve"
	"github.com/ActiveState/cli/internal/runners/manifest"
	"github.com/ActiveState/cli/internal/runners/projects"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	defer func() {
		logging.Debug("Exiting")
		if r := recover(); r != nil {
			logging.Error("Recovered from panic: %v", r)
			fmt.Printf("Recovered from panic: %v\n", r)
			os.Exit(1)
		}
	}()
	defer func() {
		if err := events.WaitForEvents(5*time.Second, logging.Close); err != nil {
			logging.Warning("Failed waiting for events: %v", err)
		}
	}()

	mcpHandler := registerServer()

	// Parse command line flags
	rawFlag := flag.Bool("raw", false, "Expose all State Tool commands as tools; this will lead to issues and is not optimized for AI use")
	flag.Parse()
	if *rawFlag {
		close := registerRawTools(mcpHandler)
		defer close()
	} else {
		registerCuratedTools(mcpHandler)
	}

	// Start the stdio server
	logging.Info("Starting MCP server")
	if err := server.ServeStdio(mcpHandler.mcpServer); err != nil {
		logging.Error("Server error: %v\n", err)
	}
}

func registerServer() *mcpServerHandler {
	ipcClient, svcPort, err := connectToSvc()
	if err != nil {
		panic(errs.JoinMessage(err))
	}

	// Create MCP server
	s := server.NewMCPServer(
		constants.CommandName,
		constants.VersionNumber,
	)

	mcpHandler := &mcpServerHandler{
		mcpServer: s,
		ipcClient: ipcClient,
		svcPort:   svcPort,
	}

	return mcpHandler
}

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

type mcpServerHandler struct {
	mcpServer *server.MCPServer
	ipcClient *ipc.Client
	svcPort   string
}

func (t *mcpServerHandler) addResource(resource mcp.Resource, handler server.ResourceHandlerFunc) {
	t.mcpServer.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("Recovered from resource handler panic: %v", r)
				fmt.Printf("Recovered from resource handler panic: %v\n", r)
			}
		}()
		logging.Debug("Received resource request: %s", resource.Name)
		r, err := handler(ctx, request)
		if err != nil {
			logging.Error("%s: Error handling resource request: %v", resource.Name, err)
			return nil, errs.Wrap(err, "Failed to handle resource request")
		}
		return r, nil
	})
}

func (t *mcpServerHandler) addTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	t.mcpServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (r *mcp.CallToolResult, rerr error) {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("Recovered from tool handler panic: %v", r)
				fmt.Printf("Recovered from tool handler panic: %v\n", r)
			}
		}()
		logging.Debug("Received tool request: %s", tool.Name)
		r, err := handler(ctx, request)
		logging.Debug("Received tool response from %s", tool.Name)
		if err != nil {
			logging.Error("%s: Error handling tool request: %v", tool.Name, errs.JoinMessage(err))
			// Format all errors as a single string, so the client gets the full context
			return nil, fmt.Errorf("%s: %s", tool.Name, errs.JoinMessage(err))
		}
		return r, nil
	})
}

func (t *mcpServerHandler) listProjectsHandler(ctx context.Context, request mcp.CallToolRequest) (r *mcp.CallToolResult, rerr error) {
	var byt bytes.Buffer
	prime, close, err := t.newPrimer("", &byt)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to create primer")
	}
	defer func() {
		if err := close(); err != nil {
			rerr = errs.Pack(rerr, err)
		}
	}()

	runner := projects.NewProjects(prime)
	params := projects.NewParams()
	err = runner.Run(params)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run projects")
	}

	return mcp.NewToolResultText(byt.String()), nil
}

func (t *mcpServerHandler) listProjectsResourceHandler(ctx context.Context, request mcp.ReadResourceRequest) (r []mcp.ResourceContents, rerr error) {
	var byt bytes.Buffer
	prime, close, err := t.newPrimer("", &byt)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to create primer")
	}
	defer func() {
		if err := close(); err != nil {
			rerr = errs.Pack(rerr, err)
		}
	}()

	runner := projects.NewProjects(prime)
	params := projects.NewParams()
	err = runner.Run(params)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run projects")
	}

	r = append(r, mcp.TextResourceContents{Text: byt.String()})
	return r, nil
}

func (t *mcpServerHandler) manifestHandler(ctx context.Context, request mcp.CallToolRequest) (r *mcp.CallToolResult, rerr error) {
	pjPath := request.Params.Arguments["project_directory"].(string)

	var byt bytes.Buffer
	prime, close, err := t.newPrimer(pjPath, &byt)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to create primer")
	}
	defer func() {
		if err := close(); err != nil {
			rerr = errs.Pack(rerr, err)
		}
	}()

	m := manifest.NewManifest(prime)
	err = m.Run(manifest.Params{})
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run manifest")
	}

	return mcp.NewToolResultText(byt.String()), nil
}

func (t *mcpServerHandler) cveHandler(ctx context.Context, request mcp.CallToolRequest) (r *mcp.CallToolResult, rerr error) {
	pjPath := request.Params.Arguments["project_directory"].(string)

	var byt bytes.Buffer
	prime, close, err := t.newPrimer(pjPath, &byt)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to create primer")
	}
	defer func() {
		if err := close(); err != nil {
			rerr = errs.Pack(rerr, err)
		}
	}()

	c := cve.NewCve(prime)
	err = c.Run(&cve.Params{})
	if err != nil {
		return nil, errs.Wrap(err, "Failed to run manifest")
	}

	return mcp.NewToolResultText(byt.String()), nil
}

func (t *mcpServerHandler) lookupCveHandler(ctx context.Context, request mcp.CallToolRequest) (r *mcp.CallToolResult, rerr error) {
	cveId := request.Params.Arguments["cve_ids"].(string)
	cveIds := strings.Split(cveId, ",")

	results, err := LookupCve(cveIds...)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to lookup CVEs")
	}

	byt, err := json.Marshal(results)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to marshal results")
	}

	return mcp.NewToolResultText(string(byt)), nil
}

type stdOutput struct{}

func (s *stdOutput) Notice(msg interface{}) {
	logging.Info(fmt.Sprintf("%v", msg))
}

func connectToSvc() (*ipc.Client, string, error) {
	svcExec, err := installation.ServiceExec()
	if err != nil {
		return nil, "", errs.Wrap(err, "Could not get service info")
	}

	ipcClient := svcctl.NewDefaultIPCClient()
	argText := strings.Join(os.Args, " ")
	svcPort, err := svcctl.EnsureExecStartedAndLocateHTTP(ipcClient, svcExec, argText, &stdOutput{})
	if err != nil {
		return nil, "", errs.Wrap(err, "Failed to start state-svc at state tool invocation")
	}

	return ipcClient, svcPort, nil
}

func (t *mcpServerHandler) newPrimer(projectDir string, o *bytes.Buffer) (*primer.Values, func() error, error) {
	closers := []func() error{}
	closer := func() error {
		for _, c := range closers {
			if err := c(); err != nil {
				return err
			}
		}
		return nil
	}

	cfg, err := config.New()
	if err != nil {
		return nil, closer, errs.Wrap(err, "Failed to create config")
	}
	closers = append(closers, cfg.Close)

	auth := authentication.New(cfg)
	closers = append(closers, auth.Close)

	out, err := output.New(string(output.SimpleFormatName), &output.Config{
		OutWriter:   o,
		ErrWriter:   o,
		Colored:     false,
		Interactive: false,
		ShellName:   "",
	})
	if err != nil {
		return nil, closer, errs.Wrap(err, "Failed to create output")
	}

	var pj *project.Project
	if projectDir != "" {
		pjf, err := projectfile.FromPath(projectDir)
		if err != nil {
			return nil, closer, errs.Wrap(err, "Failed to create projectfile")
		}
		pj, err = project.New(pjf, out)
		if err != nil {
			return nil, closer, errs.Wrap(err, "Failed to create project")
		}
	}

	// Set up conditional, which accesses a lot of primer data
	sshell := subshell.New(cfg)

	conditional := constraints.NewPrimeConditional(auth, pj, sshell.Shell())
	project.RegisterConditional(conditional)
	if err := project.RegisterExpander("mixin", project.NewMixin(auth).Expander); err != nil {
		logging.Debug("Could not register mixin expander: %v", err)
	}

	svcmodel := model.NewSvcModel(t.svcPort)

	if auth.AvailableAPIToken() != "" {
		jwt, err := svcmodel.GetJWT(context.Background())
		if err != nil {
			multilog.Critical("Could not get JWT: %v", errs.JoinMessage(err))
		}
		if err != nil || jwt == nil {
			// Could not authenticate; user got logged out
			auth.Logout()
		} else {
			auth.UpdateSession(jwt)
		}
	}

	return primer.New(pj, out, auth, sshell, conditional, cfg, t.ipcClient, svcmodel), closer, nil
}
