package cmdtree

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/state"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/state/secrets"
)

// CmdTree manages a tree of captain.Command instances.
type CmdTree struct {
	cmd *captain.Command
}

// New prepares a CmdTree.
func New(prime *primer.Values, args ...string) *CmdTree {
	globals := newGlobalOptions()

	registry := captain.NewRegistry(prime.Output())

	authCmd := newAuthCommand(registry, prime)
	authCmd.AddChildren(
		newSignupCommand(registry, prime),
		newLogoutCommand(registry, prime),
	)

	exportCmd := newExportCommand(registry, prime)
	exportCmd.AddChildren(
		newRecipeCommand(registry, prime),
		newJWTCommand(registry, prime),
		newPrivateKeyCommand(registry, prime),
		newAPIKeyCommand(registry, prime),
		newExportConfigCommand(registry, prime),
		newExportGithubActionCommand(registry, prime),
	)

	packagesCmd := newPackagesCommand(registry, prime)
	packagesCmd.AddChildren(
		newPackagesAddCommand(registry, prime),
		newPackagesUpdateCommand(registry, prime),
		newPackagesRemoveCommand(registry, prime),
		newPackagesImportCommand(registry, prime),
		newPackagesSearchCommand(registry, prime),
	)

	platformsCmd := newPlatformsCommand(registry, prime)
	platformsCmd.AddChildren(
		newPlatformsSearchCommand(registry, prime),
		newPlatformsAddCommand(registry, prime),
		newPlatformsRemoveCommand(registry, prime),
	)

	scriptsCmd := newScriptsCommand(registry, prime)
	scriptsCmd.AddChildren(
		newScriptsEditCommand(registry, prime),
	)

	languagesCmd := newLanguagesCommand(registry, prime)
	languagesCmd.AddChildren(newLanguageUpdateCommand(registry, prime))

	cleanCmd := newCleanCommand(registry, prime)
	cleanCmd.AddChildren(
		newUninstallCommand(registry, prime),
		newCacheCommand(registry, prime),
		newConfigCommand(registry, prime),
	)

	deployCmd := newDeployCommand(registry, prime)
	deployCmd.AddChildren(
		newDeployInstallCommand(registry, prime),
		newDeployConfigureCommand(registry, prime),
		newDeploySymlinkCommand(registry, prime),
		newDeployReportCommand(registry, prime),
	)

	tutorialCmd := newTutorialCommand(registry, prime)
	tutorialCmd.AddChildren(newTutorialProjectCommand(registry, prime))

	eventsCmd := newEventsCommand(registry, prime)
	eventsCmd.AddChildren(newEventsLogCommand(registry, prime))

	stateCmd := newStateCommand(registry, globals, prime)
	stateCmd.AddChildren(
		newActivateCommand(registry, prime),
		newInitCommand(registry, prime),
		newPushCommand(registry, prime),
		newProjectsCommand(registry, prime),
		authCmd,
		exportCmd,
		newOrganizationsCommand(registry, prime),
		newRunCommand(registry, prime),
		newShowCommand(registry, prime),
		packagesCmd,
		platformsCmd,
		newHistoryCommand(registry, prime),
		cleanCmd,
		languagesCmd,
		deployCmd,
		scriptsCmd,
		eventsCmd,
		newPullCommand(registry, prime),
		newUpdateCommand(registry, prime),
		newForkCommand(registry, prime),
		newPpmCommand(registry, prime),
		newInviteCommand(registry, prime),
		tutorialCmd,
		newPrepareCommand(registry, prime),
		newProtocolCommand(registry, prime),
		newShimCommand(registry, prime, args...),
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

func newStateCommand(registry *captain.Registry, globals *globalOptions, prime *primer.Values) *captain.Command {
	opts := state.NewOptions()

	runner := state.New(opts, prime)
	cmd := registry.NewCommand(
		"state",
		"",
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
				/* This option is only used for the vscode extension: It prevents the integrated terminal to close immediately after an error occurs, such that the user can read the message */
				Name:        "confirm-exit-on-error", // Name and Shorthand should be kept in sync with cmd/state/main.go
				Description: "prompts the user to press enter before exiting, when an error occurs",
				Persist:     true,
				Hidden:      true, // No need to add this to help messages
				Value:       &opts.ConfirmExit,
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

// Command returns the root command of the CmdTree
func (ct *CmdTree) Command() *captain.Command {
	return ct.cmd
}

// applyLegacyChildren will register any commands and expanders
func applyLegacyChildren(cmd *captain.Command, globals *globalOptions) {
	logging.Debug("register")

	secretsapi.InitializeClient()

	cmd.AddLegacyChildren(
		secrets.NewCommand(secretsapi.Get(), &globals.Output).Config(),
	)
}
