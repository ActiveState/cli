package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/runners/cve"
	"github.com/ActiveState/cli/internal/runners/manifest"
	"github.com/ActiveState/cli/internal/runners/projects"
	"github.com/mark3labs/mcp-go/mcp"
)

// listProjectsHandler handles the list_projects tool
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

// manifestHandler handles the view_manifest tool
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

// cveHandler handles the view_cves tool
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

// lookupCveHandler handles the lookup_cve tool
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