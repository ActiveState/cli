package pkg

import (
	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/project"
)

// SearchArgs holds the arg values passed through the command line
var SearchArgs struct {
	Name string
}

// SearchFlags holds the search-related flag values passed through the command line
var SearchFlags struct {
	Language string
}

// SearchCommand is the `packages search` command struct
var SearchCommand = &commands.Command{
	Name:        "search",
	Description: "package_search_description",

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "package_arg_name",
			Description: "package_arg_name_description",
			Variable:    &SearchArgs.Name,
			Required:    true,
		},
	},
	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "language",
			Description: "package_search_flag_language_description",
			Type:        commands.TypeString,
			StringVar:   &SearchFlags.Language,
		},
	},
}

func init() {
	SearchCommand.Run = ExecuteSearch // Work around initialization loop
}

// ExecuteSearch is executed when `state packages search` is ran
func ExecuteSearch(cmd *cobra.Command, allArgs []string) {
	logging.Debug("ExecuteSearch")

	proj := project.Get()
	_ = proj
	print.Line(SearchArgs.Name, SearchFlags.Language)
}
