package activate

import (
	"errors"
	"os"
	"sync"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/internal/scm"
	"github.com/ActiveState/ActiveState-CLI/internal/structures"
	"github.com/ActiveState/ActiveState-CLI/internal/subshell"
	"github.com/ActiveState/ActiveState-CLI/internal/virtualenvironment"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/hooks"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/ActiveState/cobra"
)

// Command holds our main command definition
var Command = &structures.Command{
	Name:        "activate",
	Description: "activate_project",
	Run:         Execute,
}

// Flags hold the flag values passed through the command line
var Flags struct {
	Path   string
	Branch string
}

func init() {
	logging.Debug("init")

	Command.GetCobraCmd().PersistentFlags().StringVar(&Flags.Path, "path", "", locale.T("flag_state_activate_path_description"))
	Command.GetCobraCmd().PersistentFlags().StringVar(&Flags.Branch, "branch", "", locale.T("flag_state_activate_branch_description"))
}

// Clones the repository specified by a given URI or ID and returns it. Any
// error that occurs during the clone process is also returned.
func clone(uriOrID string) (scm.SCMer, error) {
	scm := scm.New(uriOrID)
	if scm != nil {
		if Flags.Path != "" {
			scm.SetPath(Flags.Path)
		}
		if Flags.Branch != "" {
			scm.SetBranch(Flags.Branch)
		}
		if !scm.ConfigFileExists() {
			return nil, errors.New(locale.T("error_state_activate_config_exists"))
		}
		if err := scm.Clone(); err != nil {
			print.Error(locale.T("error_state_activate"))
			return nil, err
		}
	} else {
		return nil, errors.New("not implemented yet") // TODO: activate from ID
	}
	return scm, nil
}

// Loads the given ActiveState project configuration file and returns it as a
// struct. Any error that occurs during the clone process is also returned.
func loadProjectConfig(configFile string) (*projectfile.Project, error) {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		print.Error(locale.T("error_state_activate_config_exists", map[string]interface{}{"ConfigFile": constants.ConfigFileName}))
		return nil, err
	}
	return projectfile.Parse(configFile)
}

// Execute the activate command
func Execute(cmd *cobra.Command, args []string) {
	var wg sync.WaitGroup

	logging.Debug("Execute")
	if len(args) > 0 {
		scm, err := clone(args[0])
		if err != nil {
			print.Error(locale.T("error_cannot_clone_uri", map[string]interface{}{"URI": args[0]}))
			print.Error(err.Error())
			return
		}

		print.Info(locale.T("info_state_activate_cd", map[string]interface{}{"Dir": scm.Path()}))
		os.Chdir(scm.Path())

		if Flags.Branch != "" {
			print.Info(locale.T("info_state_activate_branch", map[string]interface{}{"Branch": scm.Branch()}))
			err = scm.CheckoutBranch()
			if err != nil {
				print.Error(locale.T("error_cannot_checkout_branch"))
				print.Error(err.Error())
				return
			}
		}
	}

	project, err := projectfile.Get()
	if err != nil {
		print.Error(locale.T("error_state_activate_config_load"))
		print.Error(err.Error())
		return
	}

	err = virtualenvironment.Activate(project)
	if err != nil {
		print.Error(locale.T("error_could_not_activate_venv"))
		print.Error(err.Error())
		return
	}

	err = hooks.RunHook("ACTIVATE", project)
	if err != nil {
		print.Error(locale.T("error_could_not_run_hooks"))
		print.Error(err.Error())
		return
	}

	venv, err := subshell.Activate(&wg)
	_ = venv

	if err != nil {
		print.Error(locale.T("error_could_not_activate_subshell"))
		print.Error(err.Error())
		return
	}

	// Don't exit until our subshell has finished
	wg.Wait()

	print.Bold(locale.T("info_deactivated", project))

}
