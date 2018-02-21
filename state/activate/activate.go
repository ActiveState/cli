package activate

import (
	"os"
	"sync"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/internal/scm"
	"github.com/ActiveState/ActiveState-CLI/internal/subshell"
	"github.com/ActiveState/ActiveState-CLI/internal/virtualenvironment"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/commands"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/hooks"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/ActiveState/cobra"
)

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "activate",
	Description: "activate_project",
	Run:         Execute,

	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "path",
			Shorthand:   "",
			Description: "flag_state_activate_path_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Path,
		},
		&commands.Flag{
			Name:        "branch",
			Shorthand:   "",
			Description: "flag_state_activate_branch_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Branch,
		},
	},

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "arg_state_activate_url",
			Description: "arg_state_activate_url_description",
			Variable:    &Args.URL,
		},
	},
}

// Flags hold the flag values passed through the command line
var Flags struct {
	Path   string
	Branch string
}

// Args hold the arg values passed through the command line
var Args struct {
	URL string
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
			return nil, failures.User.New(locale.T("error_state_activate_config_exists"))
		}
		if err := scm.Clone(); err != nil {
			print.Error(locale.T("error_state_activate"))
			return nil, err
		}
	} else {
		return nil, failures.User.New("not implemented yet") // TODO: activate from ID
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
	if Args.URL != "" {
		scm, err := clone(Args.URL)
		if err != nil {
			failures.Handle(err, locale.T("error_cannot_clone_uri", map[string]interface{}{"URI": Args.URL}))
			return
		}

		print.Info(locale.T("info_state_activate_cd", map[string]interface{}{"Dir": scm.Path()}))
		os.Chdir(scm.Path())

		if Flags.Branch != "" {
			print.Info(locale.T("info_state_activate_branch", map[string]interface{}{"Branch": scm.Branch()}))
			err = scm.CheckoutBranch()
			if err != nil {
				failures.Handle(err, locale.T("error_cannot_checkout_branch"))
				return
			}
		}
	}

	project, err := projectfile.Get()
	if err != nil {
		failures.Handle(err, locale.T("error_state_activate_config_load"))
		return
	}

	err = virtualenvironment.Activate(project)
	if err != nil {
		failures.Handle(err, locale.T("error_could_not_activate_venv"))
		return
	}

	err = hooks.RunHook("ACTIVATE", project)
	if err != nil {
		failures.Handle(err, locale.T("error_could_not_run_hooks"))
		return
	}

	venv, err := subshell.Activate(&wg)
	_ = venv

	if err != nil {
		failures.Handle(err, locale.T("error_could_not_activate_subshell"))
		return
	}

	// Don't exit until our subshell has finished
	wg.Wait()

	print.Bold(locale.T("info_deactivated", project))

}
