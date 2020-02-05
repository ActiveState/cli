package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
)

// List manages the listing execution context.
type List struct{}

// NewList prepares a list execution context for use.
func NewList() *List {
	return &List{}
}

// Run executes the list behavior.
func (l *List) Run() error {
	logging.Debug("Execute platforms list")
	return list()
}

func list() error {
	return nil
}
