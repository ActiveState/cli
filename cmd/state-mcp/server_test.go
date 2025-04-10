package main

import (
	"context"
	"encoding/json"
	"testing"
)

func TestServer(t *testing.T) {
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
