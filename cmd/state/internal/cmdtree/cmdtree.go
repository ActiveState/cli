package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runners/state"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/state/invite"
	"github.com/ActiveState/cli/state/scripts"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/ActiveState/cli/state/show"
)

// CmdTree manages a tree of captain.Command instances.
type CmdTree struct {
	cmd *captain.Command
}

// New prepares a CmdTree.
func New(pj *project.Project, outputer output.Outputer, prompter prompt.Prompter) *CmdTree {
	globals := newGlobalOptions()

	auth := authentication.Get()

	authCmd := newAuthCommand(globals)
	authCmd.AddChildren(
		newSignupCommand(),
		newLogoutCommand(),
	)

	exportCmd := newExportCommand()
	exportCmd.AddChildren(
		newRecipeCommand(),
		newJWTCommand(),
		newPrivateKeyCommand(),
		newAPIKeyCommand(outputer),
	)

	packagesCmd := newPackagesCommand(outputer)
	packagesCmd.AddChildren(
		newPackagesAddCommand(outputer),
		newPackagesUpdateCommand(outputer),
		newPackagesRemoveCommand(outputer),
		newPackagesImportCommand(outputer),
		newPackagesSearchCommand(outputer),
	)

	platformsCmd := newPlatformsCommand(outputer)
	platformsCmd.AddChildren(
		newPlatformsSearchCommand(outputer),
		newPlatformsAddCommand(outputer),
		newPlatformsRemoveCommand(outputer),
	)

	languagesCmd := newLanguagesCommand(outputer)
	languagesCmd.AddChildren(newLanguageUpdateCommand(outputer))

	cleanCmd := newCleanCommand(outputer)
	cleanCmd.AddChildren(
		newUninstallCommand(outputer),
		newCacheCommand(outputer),
		newConfigCommand(outputer),
	)

	deployCmd := newDeployCommand(outputer)
	deployCmd.AddChildren(
		newDeployInstallCommand(outputer),
		newDeployConfigureCommand(outputer),
		newDeploySymlinkCommand(outputer),
		newDeployReportCommand(outputer),
	)

	stateCmd := newStateCommand(globals)
	stateCmd.AddChildren(
		newActivateCommand(outputer),
		newInitCommand(),
		newPushCommand(),
		newProjectsCommand(outputer, auth),
		authCmd,
		exportCmd,
		newOrganizationsCommand(globals),
		newRunCommand(outputer),
		packagesCmd,
		platformsCmd,
		newHistoryCommand(outputer),
		cleanCmd,
		languagesCmd,
		deployCmd,
		newEventsCommand(pj, outputer),
		newPullCommand(pj, outputer),
		newUpdateCommand(pj, outputer),
		newForkCommand(pj, auth, outputer, prompter),
		newPpmCommand(),
	)

	applyLegacyChildren(stateCmd, globals)

	return &CmdTree{
		cmd: stateCmd,
	}
}

type globalOptions struct {
	Verbose    bool
	Output     string
	Monochrome bool
}

func newGlobalOptions() *globalOptions {
	return &globalOptions{}
}

func newStateCommand(globals *globalOptions) *captain.Command {
	opts := state.NewOptions()

	runner := state.New(opts)
	cmd := captain.NewCommand(
		"state",
		locale.T("state_description"),
		[]*captain.Flag{
			{
				Name:        "locale",
				Shorthand:   "l",
				Description: locale.T("flag_state_locale_description"),
				Persist:     true,
				Value:       &opts.Locale,
			},
			{
				Name:        "verbose",
				Shorthand:   "v",
				Description: locale.T("flag_state_verbose_description"),
				Persist:     true,
				OnUse: func() {
					if !condition.InTest() {
						logging.CurrentHandler().SetVerbose(true)
					}
				},
				Value: &globals.Verbose,
			},
			{
				Name:        "mono", // Name and Shorthand should be kept in sync with cmd/state/main.go
				Persist:     true,
				Description: locale.T("flag_state_monochrome_output_description"),
				Value:       &globals.Monochrome,
			},
			{
				Name:        "output", // Name and Shorthand should be kept in sync with cmd/state/main.go
				Shorthand:   "o",
				Description: locale.T("flag_state_output_description"),
				Persist:     true,
				Value:       &globals.Output,
			},
			{
				Name:        "version",
				Description: locale.T("flag_state_version_description"),
				Value:       &opts.Version,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			if globals.Verbose {
				logging.CurrentHandler().SetVerbose(true)
			}

			return runner.Run(ccmd.Usage)
		},
	)

	cmd.SetUsageTemplate("usage_tpl")

	return cmd
}

// Execute runs the CmdTree using the provided CLI arguments.
func (ct *CmdTree) Execute(args []string) error {
	return ct.cmd.Execute(args)
}

func setLegacyOutput(globals *globalOptions) {
	scripts.Flags.Output = &globals.Output
	show.Flags.Output = &globals.Output
}

// applyLegacyChildren will register any commands and expanders
func applyLegacyChildren(cmd *captain.Command, globals *globalOptions) {
	logging.Debug("register")

	secretsapi.InitializeClient()

	setLegacyOutput(globals)

	cmd.AddLegacyChildren(
		show.Command,
		scripts.Command,
		invite.Command,
		secrets.NewCommand(secretsapi.Get(), &globals.Output).Config(),
	)
}
