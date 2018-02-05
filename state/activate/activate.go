package activate

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
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
			err := scm.Clone()
			if err != nil {
				print.Error(locale.T("error_state_activate"))
				return // TODO: how to return error?
			}
		} else {
			return // TODO: activate from ID
		}
		configFile := filepath.Join(scm.Path(), constants.ConfigFileName)
		if Flags.Cd {
			print.Info(locale.T("info_state_activate_cd", map[string]interface{}{"Dir": scm.Path()}))
			os.Chdir(scm.Path())
			configFile = constants.ConfigFileName
		}
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			print.Error(locale.T("error_state_activate_config", map[string]interface{}{"ConfigFile": constants.ConfigFileName}))
			return // TODO: how to return error?
		}
	} else {
		// TODO: activate current directory
		// scm := scm.New(os.Getwd())
	}
}
