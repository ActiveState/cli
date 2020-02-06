package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
)

// Add manages the adding execution context.
type Add struct{}

// NewAdd prepares an add execution context for use.
func NewAdd() *Add {
	return &Add{}
}

// Run executes the add behavior.
func (a *Add) Run() error {
	logging.Debug("Execute platforms add")

	return add()
}

func add() error {
	return nil
}
