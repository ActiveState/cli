package activate

import (
	"os"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/internal/scm"
	"github.com/ActiveState/ActiveState-CLI/internal/structures"
	"github.com/ActiveState/cobra"
	"github.com/dvirsky/go-pylog/logging"
)

// Command holds our main command definition
var Command = &structures.Command{
	Name:        "activate",
	Description: "activate_project",
	Run:         Execute,
}

// Flags hold the flag values passed through the command line
var Flags struct {
	Path string
	Cd   bool
}

func init() {
	logging.Debug("init")

	Command.GetCobraCmd().PersistentFlags().StringVar(&Flags.Path, "path", "", locale.T("flag_state_activate_path_description"))
	Command.GetCobraCmd().PersistentFlags().BoolVar(&Flags.Cd, "cd", false, locale.T("flag_state_activate_cd_description"))
}

// Execute the activate command
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
	if len(args) > 0 {
		scm := scm.New(args[0])
		if scm != nil {
			if Flags.Path != "" {
				scm.SetPath(Flags.Path)
			}
			scm.Clone()
		} else {
			// TODO: activate from ID
		}
		if Flags.Cd {
			print.Info(locale.T("info_state_activate_cd", map[string]interface{}{"Dir": scm.Path()}))
			os.Chdir(scm.Path())
		}
	} else {
		// TODO: activate current directory
		// scm := scm.New(os.Getwd())
	}
}
