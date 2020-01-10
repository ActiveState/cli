package export

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/logging"
)

type Export struct{}

func NewExport() *Export {
	return &Export{}
}

func (e *Export) Run(cmd *captain.Command) error {
	logging.Debug("Execute")
	err := cmd.Usage()
	if err != nil {
		return err
	}
	return nil
}
