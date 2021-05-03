package cmdtree

import (
	"github.com/ActiveState/cli/cmd/state/internal/cmdtree/intercepts/cmdcall"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/state"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/project"
)

// CmdTree manages a tree of captain.Command instances.
type CmdTree struct {
	cmd *captain.Command
}

// New prepares a CmdTree.
func New(prime *primer.Values, args ...string) *CmdTree {
	globals := newGlobalOptions()

	authCmd := newAuthCommand(prime)
	authCmd.AddChildren(
		newSignupCommand(prime),
		newLogoutCommand(prime),
	)

	cveCmd := newCveCommand(prime)
	cveCmd.AddChildren(
		newReportCommand(prime),
		newOpenCommand(prime),
	)

	exportCmd := newExportCommand(prime)
	exportCmd.AddChildren(
		newRecipeCommand(prime),
		newJWTCommand(prime),
		newPrivateKeyCommand(prime),
		newAPIKeyCommand(prime),
		newExportConfigCommand(prime),
		newExportGithubActionCommand(prime),
		newExportDocsCommand(prime),
	)

	platformsCmd := newPlatformsCommand(prime)
	platformsCmd.AddChildren(
		newPlatformsSearchCommand(prime),
		newPlatformsAddCommand(prime),
		newPlatformsRemoveCommand(prime),
	)

	scriptsCmd := newScriptsCommand(prime)
	scriptsCmd.AddChildren(
		newScriptsEditCommand(prime),
	)

	languagesCmd := newLanguagesCommand(prime)
	languagesCmd.AddChildren(newLanguageInstallCommand(prime))

	cleanCmd := newCleanCommand(prime)
	cleanCmd.AddChildren(
		newCleanUninstallCommand(prime),
		newCleanCacheCommand(prime),
		newCleanConfigCommand(prime),
	)

	deployCmd := newDeployCommand(prime)
	deployCmd.AddChildren(
		newDeployInstallCommand(prime),
		newDeployConfigureCommand(prime),
		newDeploySymlinkCommand(prime),
		newDeployReportCommand(prime),
	)

	tutorialCmd := newTutorialCommand(prime)
	tutorialCmd.AddChildren(newTutorialProjectCommand(prime))

	eventsCmd := newEventsCommand(prime)
	eventsCmd.AddChildren(newEventsLogCommand(prime))

	installCmd := newInstallCommand(prime)
	uninstallCmd := newUninstallCommand(prime)
	importCmd := newImportCommand(prime)
	searchCmd := newSearchCommand(prime)
	infoCmd := newInfoCommand(prime)

	pkgsCmd := newPackagesCommand(prime)
	addAs := addCmdAs{
		pkgsCmd,
		prime,
	}
	addAs.deprecatedAlias(installCmd, "add")
	addAs.deprecatedAlias(installCmd, "update")
	addAs.deprecatedAlias(uninstallCmd, "remove")
	addAs.deprecatedAlias(importCmd, "import")
	addAs.deprecatedAlias(searchCmd, "search")

	bundlesCmd := newBundlesCommand(prime)
	bundlesCmd.AddChildren(
		newBundleInstallCommand(prime),
		newBundleUninstallCommand(prime),
		newBundlesSearchCommand(prime),
	)

	secretsClient := secretsapi.InitializeClient()
	secretsCmd := newSecretsCommand(secretsClient, prime)
	secretsCmd.AddChildren(
		newSecretsGetCommand(prime),
		newSecretsSetCommand(prime),
		newSecretsSyncCommand(secretsClient, prime),
	)

	projectsCmd := newProjectsCommand(prime)
	projectsCmd.AddChildren(newRemoteProjectsCommand(prime))

	updateCmd := newUpdateCommand(prime)
	updateCmd.AddChildren(newUpdateLockCommand(prime))

	branchCmd := newBranchCommand(prime)
	branchCmd.AddChildren(
		/*  Disabled as per https://www.pivotaltracker.com/story/show/177051006
		newBranchAddCommand(prime),
		*/
		newBranchSwitchCommand(prime),
	)
	prepareCmd := newPrepareCommand(prime)
	prepareCmd.AddChildren(newPrepareCompletionsCommand(prime))

	stateCmd := newStateCommand(globals, prime)
	stateCmd.AddChildren(
		newActivateCommand(prime),
		newInitCommand(prime),
		newPushCommand(prime),
		cveCmd,
		projectsCmd,
		authCmd,
		exportCmd,
		newOrganizationsCommand(prime),
		newRunCommand(prime),
		newShowCommand(prime),
		installCmd,
		uninstallCmd,
		importCmd,
		searchCmd,
		infoCmd,
		pkgsCmd,
		bundlesCmd,
		platformsCmd,
		newHistoryCommand(prime),
		cleanCmd,
		languagesCmd,
		deployCmd,
		scriptsCmd,
		eventsCmd,
		newPullCommand(prime),
		updateCmd,
		newForkCommand(prime),
		newPpmCommand(prime),
		newInviteCommand(prime),
		tutorialCmd,
		prepareCmd,
		newProtocolCommand(prime),
		newExecCommand(prime, args...),
		newRevertCommand(prime),
		secretsCmd,
		branchCmd,
	)

	return &CmdTree{
		cmd: stateCmd,
	}
}

type globalOptions struct {
	Verbose        bool
	Output         string
	Monochrome     bool
	NonInteractive bool
}

// Group instances are used to group command help output.
var (
	EnvironmentGroup = captain.NewCommandGroup(locale.Tl("group_environment", "Environment Management"), 10)
	PackagesGroup    = captain.NewCommandGroup(locale.Tl("group_packages", "Package Management"), 9)
	PlatformGroup    = captain.NewCommandGroup(locale.Tl("group_tools", "Platform"), 8)
	VCSGroup         = captain.NewCommandGroup(locale.Tl("group_vcs", "Version Control"), 7)
	AutomationGroup  = captain.NewCommandGroup(locale.Tl("group_automation", "Automation"), 6)
	UtilsGroup       = captain.NewCommandGroup(locale.Tl("group_utils", "Utilities"), 5)
)

func newGlobalOptions() *globalOptions {
	return &globalOptions{}
}

func newStateCommand(globals *globalOptions, prime *primer.Values) *captain.Command {
	opts := state.NewOptions()

	runner := state.New(opts, prime)
	cmd := captain.NewCommand(
		"state",
		"",
		locale.T("state_description"),
		prime.Output(),
		prime.Config(),
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
				Name:        "mono", // Name and Shorthand should be kept in sync with cmd/state/output.go
				Persist:     true,
				Description: locale.T("flag_state_monochrome_output_description"),
				Value:       &globals.Monochrome,
			},
			{
				Name:        "output", // Name and Shorthand should be kept in sync with cmd/state/output.go
				Shorthand:   "o",
				Description: locale.T("flag_state_output_description"),
				Persist:     true,
				Value:       &globals.Output,
			},
			{
				/* This option is only used for the vscode extension: It prevents the integrated terminal to close immediately after an error occurs, such that the user can read the message */
				Name:        "confirm-exit-on-error", // Name and Shorthand should be kept in sync with cmd/state/output.go
				Description: "prompts the user to press enter before exiting, when an error occurs",
				Persist:     true,
				Hidden:      true, // No need to add this to help messages
				Value:       &opts.ConfirmExit,
			},
			{
				Name:      "non-interactive", // Name and Shorthand should be kept in sync with cmd/state/output.go
				Shorthand: "n",
				Persist:   true,
				Value:     &globals.NonInteractive,
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

	cmdCall := cmdcall.New(prime)

	cmd.SetInterceptChain(cmdCall.InterceptExec, interceptAddHeadlessNotify(prime.Output(), prime.Project()))

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

type addCmdAs struct {
	parent *captain.Command
	prime  *primer.Values
}

func (a *addCmdAs) deprecatedAlias(aliased *captain.Command, name string) {
	cmd := captain.NewCommand(
		name,
		aliased.Title(),
		aliased.Description(),
		a.prime.Output(),
		a.prime.Config(),
		aliased.Flags(),
		aliased.Arguments(),
		func(c *captain.Command, args []string) error {
			msg := locale.Tl(
				"cmd_deprecated_notice",
				"This command is deprecated. Please use `state {{.V0}}` instead.",
				aliased.Name(),
			)

			a.prime.Output().Notice(msg)

			return aliased.ExecuteFunc()(c, args)
		},
	)

	cmd.SetHidden(true)

	a.parent.AddChildren(cmd)
}

func interceptAddHeadlessNotify(out output.Outputer, pj *project.Project) captain.InterceptFunc {
	return func(next captain.ExecuteFunc) captain.ExecuteFunc {
		return func(cmd *captain.Command, args []string) error {
			if pj != nil && pj.IsHeadless() && (cmd.Group() == PackagesGroup || cmd.Name() == activateCmdName) {
				out.Notice(output.Heading(locale.Tr("headless_notify_title")))
				out.Notice(locale.Tr("headless_notify", pj.URL(), constants.DocumentationURLHeadless))
			}

			return next(cmd, args)
		}
	}
}
