package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
)

// List manages the listing execution context.
type List struct {
	printer Printer
}

// NewList prepares a list execution context for use.
func NewList(p Printer) *List {
	return &List{
		printer: p,
	}
}

// Run executes the list behavior.
func (l *List) Run() error {
	logging.Debug("Execute platforms list")

	return list(l.printer)
}

func list(printer Printer) error {
	printer.Print("this is some info")
	return nil
}
