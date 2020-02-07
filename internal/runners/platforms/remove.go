package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RunRemoveParams tracks the info required for running Remove.
type RunRemoveParams struct {
	Name    string
	Version string
}

// Remove manages the removeing execution context.
type Remove struct{}

// NewRemove prepares a remove execution context for use.
func NewRemove() *Remove {
	return &Remove{}
}

// Run executes the remove behavior.
func (r *Remove) Run(params RunRemoveParams) error {
	logging.Debug("Execute platforms remove")

	return remove(params.Name, params.Version)
}

func remove(name, version string) error {
	proj := project.Get()

	op := model.OperationRemoved

	return model.CommitPlatform(proj.Owner(), proj.Name(), op, name, version)
}
