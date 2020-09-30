package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/shim"
)

func newShimCommand(prime *primer.Values) *captain.Command {
	runner := shim.New(prime)

	params := shim.Params{}
	cmd := captain.NewCommand(
		"shim",
		locale.T("shim_description"),
		[]*captain.Flag{
			{
				Name:        locale.T("flag_state_shim_language"),
				Description: locale.T("flag_state_shim_language_description"),
				Value:       &params.Language,
			},
		},
		[]*captain.Argument{
			{
				Name:        locale.T("args_state_shim_script"),
				Description: locale.T("args_state_shim_description"),
				Required:    true,
				Value:       &params.Script,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run(params, args...)
		},
	)
	cmd.SetSkipUpdate(true)
	cmd.SetSkipDeprecationCheck(true)

	return cmd
}
