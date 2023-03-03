package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/learn"
)

func newLearnCommand(prime *primer.Values) *captain.Command {
	learnRunner := learn.New(prime)
	var xStr captain.NullString
	var xInt captain.NullInt
	var xBool captain.NullBool

	return captain.NewCommand(
		"learn",
		locale.Tl("learn_title", "Learn about the State Tool"),
		locale.Tl("learn_description", "Read the State Tool cheat sheet to learn about common commands"),
		prime,
		[]*captain.Flag{
			{
				Name:        "str",
				Description: "description for str",
				Value:       &xStr,
			},
			{
				Name:        "num",
				Description: "description for num",
				Value:       &xInt,
			},
			{
				Name:        "tru",
				Description: "description for tru",
				Value:       &xBool,
			},
		},
		[]*captain.Argument{},
		func(cmd *captain.Command, args []string) error {
			return learnRunner.Run(xStr.AsPtrTo(), xInt.AsPtrTo(), xBool.AsPtrTo())
		}).SetGroup(UtilsGroup)
}
