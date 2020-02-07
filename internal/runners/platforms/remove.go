package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
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

	commit := &commitOp{}

	return remove(commit, params.Name, params.Version)
}

func remove(c committer, name, version string) error {
	return c.CommitPlatform(model.OperationAdded, name, version)
}
