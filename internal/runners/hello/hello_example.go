package hello

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
)

type primeable interface {
	primer.Outputer
}

type RunParams struct {
	Named string
}

func NewRunParams() *RunParams {
	return &RunParams{
		Named: "Friend",
	}
}

type Hello struct {
	out output.Outputer
}

func New(p primeable) *Hello {
	return &Hello{
		out: p.Output(),
	}
}

func (h *Hello) Run(params *RunParams) error {
	if err := runbits.SayHello(h.out, params.Named); err != nil {
		return locale.WrapError(
			err, "hello_cannot_say", "Cannot say hello without a name",
		)
	}

	return nil
}
