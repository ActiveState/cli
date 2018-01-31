package activate

import (
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/internal/structures"
	"github.com/ActiveState/cobra"
	"github.com/dvirsky/go-pylog/logging"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Command holds our main command definition
var Command = &structures.Command{
	Name:        "activate",
	Description: "activate_project",
	Run:         Execute,
}

// Command line options
var path string
var cd bool

func init() {
	logging.Debug("Init")
	Command.GetCobraCmd().PersistentFlags().StringVar(&path, "path", "", locale.T("flag_state_activate_path_description"))
	Command.GetCobraCmd().PersistentFlags().BoolVar(&cd, "cd", false, locale.T("flag_state_activate_cd_description"))
}

// Execute the activate command
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
	if len(args) > 0 {
		if strings.HasPrefix(args[0], "http://") || strings.HasPrefix(args[0], "https://") {
			// Clone from URL
			if path == "" {
				// Determine "humanish" path to clone to.
				// Based on git's clone shell script.
				re := regexp.MustCompile(":*/*\\.git$")
				path = re.ReplaceAllString(strings.TrimRight(args[0], "/"), "")
				re = regexp.MustCompile(".*[/:]")
				path = re.ReplaceAllString(path, "")
				logging.Debug("Determined 'humanish' path to be %s", path)
			}
			print.Info(locale.T("info_state_activate_url"), args[0], path)
			cmd := exec.Command("git", "clone", args[0], path)
			logging.Debug("Executing command: %s", cmd.Args)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return
			}
		} else {
			// TODO: Clone from ID
			print.Info(locale.T("info_state_activate_id"), args[0])
			logging.Debug("Activating ID")
		}
		if cd {
			os.Chdir(path)
		}
	} else {
		logging.Debug("Activating project under cwd.")
	}
}
