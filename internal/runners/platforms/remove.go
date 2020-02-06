package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
)

// Remove manages the removeing execution context.
type Remove struct{}

// NewRemove prepares a remove execution context for use.
func NewRemove() *Remove {
	return &Remove{}
}

// Run executes the remove behavior.
func (r *Remove) Run() error {
	logging.Debug("Execute platforms remove")

	return remove()
}

func remove() error {
	return nil
}
