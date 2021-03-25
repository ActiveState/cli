package model

import "github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"

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

// buildEngineFromResponse handles a headchef build status response and returns
// the relevant engine.
func buildEngineFromResponse(resp *headchef_models.BuildStatusResponse) BuildEngine {
	if resp == nil || resp.BuildEngine == nil {
		return UnknownEngine
	}

	return ParseBuildEngine(*resp.BuildEngine)
}

func (be BuildEngine) String() string {
	switch be {
	case Camel:
		return headchef_models.BuildStatusResponseBuildEngineCamel
	case Alternative:
		return headchef_models.BuildStatusResponseBuildEngineAlternative
	case Hybrid:
		return headchef_models.BuildStatusResponseBuildEngineHybrid
	default:
		return "unknown"
	}
}

func ParseBuildEngine(be string) BuildEngine {
	switch be {
	case headchef_models.BuildStatusResponseBuildEngineAlternative:
		return Alternative
	case headchef_models.BuildStatusResponseBuildEngineCamel:
		return Camel
	case headchef_models.BuildStatusResponseBuildEngineHybrid:
		return Hybrid
	default:
		return UnknownEngine
	}
}
