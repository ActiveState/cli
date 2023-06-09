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
)

// buildEngineFromResponse handles a headchef build status response and returns
// the relevant engine.
func buildEngineFromResponse(resp *headchef_models.V1BuildStatusResponse) BuildEngine {
	if resp == nil || resp.BuildEngine == nil {
		return UnknownEngine
	}

	return ParseBuildEngine(*resp.BuildEngine)
}

func (be BuildEngine) String() string {
	switch be {
	case Camel:
		return headchef_models.V1BuildStatusResponseBuildEngineCamel
	case Alternative:
		return headchef_models.V1BuildStatusResponseBuildEngineAlternative
	default:
		return "unknown"
	}
}

func ParseBuildEngine(be string) BuildEngine {
	switch be {
	case headchef_models.V1BuildStatusResponseBuildEngineAlternative:
		return Alternative
	case headchef_models.V1BuildStatusResponseBuildEngineCamel:
		return Camel
	default:
		return UnknownEngine
	}
}
