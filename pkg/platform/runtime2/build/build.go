// Package build is the top-level package for all runtime implementations.
// Currently, the ActiveState Platform supports two build engines (Camel and
// Alternative), their implementations can be found in sub-packages relative to
// this directory.
package build

import (
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
)

// BuildEngine describes the build engine that was used to build the runtime
type BuildEngine int

const (
	// UnknownEngine represents an engine unknown to the runtime.
	UnknownEngine BuildEngine = iota

	// Camel is the legacy build engine, that builds Active{Python,Perl,Tcl}
	// distributions
	Camel

	// Alternative is the new alternative build orchestration framework
	Alternative

	// Hybrid wraps Camel.
	Hybrid
)

// BuildEngineFromResponse handles a headchef build status response and returns
// the relevant engine.
func BuildEngineFromResponse(resp *headchef_models.BuildStatusResponse) BuildEngine {
	if resp == nil || resp.BuildEngine == nil {
		return UnknownEngine
	}

	switch *resp.BuildEngine {
	case headchef_models.BuildStatusResponseBuildEngineCamel:
		return Camel
	case headchef_models.BuildStatusResponseBuildEngineAlternative:
		return Alternative
	case headchef_models.BuildStatusResponseBuildEngineHybrid:
		return Hybrid
	default:
		return UnknownEngine
	}
}

// BuildResult is the unified response of a Build request
type BuildResult struct {
	BuildEngine         BuildEngine
	Recipe              *inventory_models.Recipe
	BuildStatusResponse *headchef_models.BuildStatusResponse
	BuildStatus         headchef.BuildStatusEnum
}
