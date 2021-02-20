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

// BuildResult is the unified response of a Build request
type BuildResult struct {
	BuildEngine         BuildEngine
	Recipe              *inventory_models.Recipe
	BuildStatusResponse *headchef_models.BuildStatusResponse
	BuildStatus         headchef.BuildStatusEnum
}
