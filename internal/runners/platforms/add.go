package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RunAddParams tracks the info required for running Add.
type RunAddParams struct {
	Name    string
	Version string
}

// Add manages the adding execution context.
type Add struct{}

// NewAdd prepares an add execution context for use.
func NewAdd() *Add {
	return &Add{}
}

// Run executes the add behavior.
func (a *Add) Run(params RunAddParams) error {
	logging.Debug("Execute platforms add")

	return add(params.Name, params.Version)
}

func add(name, version string) error {
	proj := project.Get()

	op := model.OperationAdded

	return model.CommitPlatform(proj.Owner(), proj.Name(), op, name, version)
}
