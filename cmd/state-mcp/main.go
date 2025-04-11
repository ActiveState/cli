package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/ActiveState/cli/internal/events"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/mark3labs/mcp-go/server"
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

	mcpHandler := registerServer()

	// Parse command line flags
	rawFlag := flag.String("type", "", "Type of MCP server to run; raw, curated or scripts")
	flag.Parse()
	switch *rawFlag {
	case "raw":
		close := registerRawTools(mcpHandler)
		defer close()
	case "scripts":
		close := registerScriptTools(mcpHandler)
		defer close()
	default:
		registerCuratedTools(mcpHandler)
	}

	// Start the stdio server
	logging.Info("Starting MCP server")
	if err := server.ServeStdio(mcpHandler.mcpServer); err != nil {
		logging.Error("Server error: %v\n", err)
	}
} 