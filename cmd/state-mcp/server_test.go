package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/cmd/state-mcp/internal/registry"
	"github.com/ActiveState/cli/internal/logging"
)

func TestServerHello(t *testing.T) {
	t.Skip(`
Fails due to state-svc not being detected when run as regular test, 
works when running with debugger. Problem for another day.
`)
	logging.CurrentHandler().SetVerbose(true)
	mcpHandler := setupServer(string(registry.CategoryDebug))
	msg := mcpHandler.Server.HandleMessage(context.Background(), json.RawMessage(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "hello",
			"arguments": {
				"name": "World"
			}
		}
	}`))
	t.Fatalf("%+v", msg)
}
