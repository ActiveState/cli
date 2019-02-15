package activate

import (
	"flag"
	"os"
	"sync"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/scm"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cobra"
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
	scm := scm.FromRemote(uriOrID)
	if scm != nil {
		if Flags.Path != "" {
			scm.SetPath(Flags.Path)
		}
		if Flags.Branch != "" {
			scm.SetBranch(Flags.Branch)
		}
		if scm.TargetExists() {
			print.Info(locale.T("info_state_active_repoexists", map[string]interface{}{"Path": scm.Path()}))
			return scm, nil
		}
		if err := scm.Clone(); err != nil {
			print.Error(locale.T("error_state_activate"))
			return nil, err
		}
	} else {
		return nil, failures.FailUser.New("activating from ID is not implemented yet") // TODO: activate from ID
	}
	return scm, nil
}

// Execute the activate command
func Execute(cmd *cobra.Command, args []string) {
	updater.PrintUpdateMessage()

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

	project := projectfile.Get()
	print.Info(locale.T("info_activating_state", project))
	var fail = virtualenvironment.Activate()
	if fail != nil {
		failures.Handle(fail, locale.T("error_could_not_activate_venv"))
		return
	}

	_, err := subshell.Activate(&wg)
	if err != nil {
		failures.Handle(err, locale.T("error_could_not_activate_subshell"))
		return
	}

	// Don't exit until our subshell has finished
	if flag.Lookup("test.v") == nil {
		wg.Wait()
	}

	print.Bold(locale.T("info_deactivated", project))

}
