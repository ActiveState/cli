package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/predict"
)

func newPredictCmd(prime *primer.Values) *captain.Command {
	runner := predict.New(prime)

	var name string

	cmd := captain.NewCommand(
		"predict",
		"",
		locale.Tl("predict_description", "predicts build success for package"),
		prime.Output(),
		prime.Config(),
		nil,
		[]*captain.Argument{
			{
				Name:        locale.Tl("arg_state_predict_name", "package"),
				Description: locale.Tl("arg_state_predict_name_description", "package for which build success should be predicted"),
				Value:       &name,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			params := predict.PredictParams{args}

			return runner.Run(params)
		},
	)

	cmd.SetGroup(EnvironmentGroup)
	return cmd
}
