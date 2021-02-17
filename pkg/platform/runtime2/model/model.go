package model

import (
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
)

// BuildResult is the unified response of a Build request
type BuildResult struct {
	BuildEngine build.BuildEngine
	Recipe      *inventory_models.Recipe
}
