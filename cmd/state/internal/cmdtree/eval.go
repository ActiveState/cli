package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/eval"
)

func newEvalCommand(prime *primer.Values) *captain.Command {
	runner := eval.New(prime)
	params := &eval.Params{}

	cmd := captain.NewCommand(
		"eval",
		"",
		locale.Tl("eval_description", "Evaluate a buildscript target"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "target",
				Description: locale.Tl("eval_args_target_description", "The target to evaluate"),
				Value:       &params.Target,
				Required:    true,
			},
		},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)
	cmd.SetGroup(AuthorGroup)

	return cmd
}
