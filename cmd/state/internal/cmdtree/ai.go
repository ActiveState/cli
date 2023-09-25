package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/ai"
)

func newAICommand(prime *primer.Values) *captain.Command {
	return captain.NewCommand(
		"ai",
		"",
		locale.Tl("ai_description", "Use AI to manage your dependencies"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			return cmd.Usage()
		}).SetGroup(UtilsGroup).SetDoesNotSupportStructuredOutput()
}

func newAISearchCommand(prime *primer.Values) *captain.Command {
	runner := ai.New(prime)
	params := ai.Params{}

	return captain.NewCommand(
		"search",
		"",
		locale.Tl("ai_description", "Search for packages using natural language"),
		prime,
		[]*captain.Flag{
			{
				Name:        "model",
				Description: locale.Tl("ai_model_description", "Which GPT model to use"),
				Value:       &params.GptModel,
			},
		},
		[]*captain.Argument{
			{
				Name:           "query",
				Description:    locale.Tl("ai_query_description", "The query to search for"),
				Value:          &params.Query,
				VariableLength: true,
			},
		},
		func(cmd *captain.Command, args []string) error {
			return runner.Run(&params)
		}).
		SetGroup(UtilsGroup).
		SetDoesNotSupportStructuredOutput().
		SetHasVariableArguments()
}
