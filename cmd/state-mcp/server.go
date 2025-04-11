package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// mcpServerHandler wraps the MCP server and provides methods for adding tools and resources
type mcpServerHandler struct {
	mcpServer *server.MCPServer
	ipcClient *ipc.Client
	svcPort   string
}

// registerServer creates and configures a new MCP server
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

// addResource adds a resource to the MCP server with error handling and logging
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

// addTool adds a tool to the MCP server with error handling and logging
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

type stdOutput struct{}

func (s *stdOutput) Notice(msg interface{}) {
	logging.Info(fmt.Sprintf("%v", msg))
}

// connectToSvc connects to the state service and returns an IPC client
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