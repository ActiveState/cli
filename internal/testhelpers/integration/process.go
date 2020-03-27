package integration

import (
	"errors"

	"github.com/ActiveState/cli/pkg/conproc"
)

type processOptions struct {
	conproc.Options
	cleanUp func() error
}

type Process struct {
	*conproc.ConsoleProcess
	pOpts processOptions
}

func (p *Process) Close() error {
	var errMsg string
	var sep string

	if err := p.pOpts.cleanUp(); err != nil {
		errMsg += err.Error()
		sep = ", "
	}

	if err := p.ConsoleProcess.Close(); err != nil {
		errMsg += sep + err.Error()
	}

	if errMsg != "" {
		return errors.New(errMsg)
	}

	return nil
}
