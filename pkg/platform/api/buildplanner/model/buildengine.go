package model

import (
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

func ParseBuildEngine(be string) types.BuildEngine {
	switch be {
	case types.Alternative.String():
		return types.Alternative
	case types.Camel.String():
		return types.Camel
	default:
		return types.UnknownEngine
	}
}
