package export

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/primer"
)

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Configurer
	primer.Projecter
	primer.Analyticer
	primer.SvcModeler
	primer.CheckoutInfoer
}

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
