package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ActiveState/cli/cmd/state-mcp/internal/mcpserver"
	"github.com/ActiveState/cli/cmd/state-mcp/internal/registry"
	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/logging"
)

func main() {
	defer func() {
		logging.Debug("Exiting")
		if r := recover(); r != nil {
			logging.Error("Recovered from panic: %v", r)
			fmt.Printf("Recovered from panic: %v, stack: %s\n", r, string(debug.Stack()))
			os.Exit(1)
		}
	}()
	defer func() {
		if err := events.WaitForEvents(5*time.Second, logging.Close); err != nil {
			logging.Warning("Failed waiting for events: %v", err)
		}
	}()

	// Parse command line flags
	rawFlag := flag.String("categories", "", "Comma separated list of categories to register tools for")
	flag.Parse()

	mcps := setupServer(strings.Split(*rawFlag, ",")...)

	// Start the stdio server
	logging.Info("Starting MCP server")
	if err := mcps.ServeStdio(); err != nil {
		logging.Error("Server error: %v\n", err)
	}
}

func setupServer(categories ...string) *mcpserver.Handler {
	mcps := mcpserver.New(newPrimer)

	registry := registry.New()
	tools := registry.GetTools(categories...)
	for _, tool := range tools {
		mcps.AddTool(tool.Tool, tool.Handler)
	}

	prompts := registry.GetPrompts(categories...)
	for _, prompt := range prompts {
		mcps.AddPrompt(prompt.Prompt, prompt.Handler)
	}

	return mcps
}
