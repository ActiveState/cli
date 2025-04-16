package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
)

func TestServerProjects(t *testing.T) {
	t.Skip("Intended for manual testing")
	mcpHandler := registerServer()
	registerRawTools(mcpHandler)
	
	msg := mcpHandler.mcpServer.HandleMessage(context.Background(), json.RawMessage(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "projects",
			"arguments": {}
		}
	}`))
	t.Fatalf("%+v", msg)
}

func TestServerPackages(t *testing.T) {
	t.Skip("Intended for manual testing")
	mcpHandler := registerServer()
	registerRawTools(mcpHandler)
	
	msg := mcpHandler.mcpServer.HandleMessage(context.Background(), json.RawMessage(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "packages",
			"arguments": {
				"project_directory": "`+environment.GetRootPathUnsafe()+`"
			}
		}
	}`))
	t.Fatalf("%+v", msg)
}
