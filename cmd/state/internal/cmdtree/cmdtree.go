package cmdtree

import (
	"time"

	"github.com/ActiveState/cli/cmd/state/internal/cmdtree/exechandlers/cmdcall"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/runners/state"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
)

// CmdTree manages a tree of captain.Command instances.
type CmdTree struct {
	cmd *captain.Command
}

// New prepares a CmdTree.
func New(prime *primer.Values, args ...string) *CmdTree {
	defer profile.Measure("cmdtree:New", time.Now())

	globals := newGlobalOptions()

	authCmd := newAuthCommand(prime, globals)
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
		newJWTCommand(prime),
		newPrivateKeyCommand(prime),
		newAPIKeyCommand(prime),
		newExportConfigCommand(prime),
		newExportGithubActionCommand(prime),
		newExportDocsCommand(prime),
		newExportEnvCommand(prime),
		newLogCommand(prime),
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
	languagesCmd.AddChildren(
		newLanguageInstallCommand(prime),
		newLanguageSearchCommand(prime),
	)

	cleanCmd := newCleanCommand(prime)
	cleanCmd.AddChildren(
		newCleanUninstallCommand(prime, globals),
		newCleanCacheCommand(prime, globals),
		newCleanConfigCommand(prime),
	)

	deployCmd := newDeployCommand(prime)
	deployCmd.AddChildren(
		newDeployInstallCommand(prime),
		newDeployConfigureCommand(prime),
		newDeploySymlinkCommand(prime),
		newDeployReportCommand(prime),
		newDeployUninstallCommand(prime),
	)

	tutorialCmd := newTutorialCommand(prime)
	tutorialCmd.AddChildren(newTutorialProjectCommand(prime))

	eventsCmd := newEventsCommand(prime)
	eventsCmd.AddChildren(newEventsLogCommand(prime))

	installCmd := newInstallCommand(prime)
	uninstallCmd := newUninstallCommand(prime)
	importCmd := newImportCommand(prime, globals)
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

	secretsClient := secretsapi.InitializeClient(prime.Auth())
	secretsCmd := newSecretsCommand(secretsClient, prime)
	secretsCmd.AddChildren(
		newSecretsGetCommand(prime),
		newSecretsSetCommand(prime),
		newSecretsSyncCommand(secretsClient, prime),
	)

	projectsCmd := newProjectsCommand(prime)
	projectsCmd.AddChildren(
		newRemoteProjectsCommand(prime),
		newProjectsEditCommand(prime),
		newDeleteProjectsCommand(prime),
		newMoveProjectsCommand(prime),
	)

	updateCmd := newUpdateCommand(prime)
	updateCmd.AddChildren(
		newUpdateLockCommand(prime, globals),
		newUpdateUnlockCommand(prime, globals))

	branchCmd := newBranchCommand(prime)
	branchCmd.AddChildren(
		/*  Disabled as per https://www.pivotaltracker.com/story/show/177051006
		newBranchAddCommand(prime),
		*/
		newBranchSwitchCommand(prime),
	)
	prepareCmd := newPrepareCommand(prime)
	prepareCmd.AddChildren(newPrepareCompletionsCommand(prime))

	configCmd := newConfigCommand(prime)
	configCmd.AddChildren(newConfigGetCommand(prime), newConfigSetCommand(prime))

	checkoutCmd := newCheckoutCommand(prime)

	useCmd := newUseCommand(prime)
	useCmd.AddChildren(
		newUseResetCommand(prime, globals),
		newUseShowCommand(prime),
	)

	shellCmd := newShellCommand(prime)

	refreshCmd := newRefreshCommand(prime)

	artifactsCmd := newArtifactsCommand(prime)
	artifactsCmd.AddChildren(
		newArtifactsDownloadCommand(prime),
	)

	stateCmd := newStateCommand(globals, prime)
	stateCmd.AddChildren(
		newHelloCommand(prime),
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
		newPullCommand(prime, globals),
		updateCmd,
		newForkCommand(prime),
		newPpmCommand(prime),
		newInviteCommand(prime),
		tutorialCmd,
		prepareCmd,
		newProtocolCommand(prime),
		newExecCommand(prime, args...),
		newRevertCommand(prime, globals),
		newResetCommand(prime, globals),
		secretsCmd,
		branchCmd,
		newLearnCommand(prime),
		configCmd,
		checkoutCmd,
		useCmd,
		shellCmd,
		refreshCmd,
		newSwitchCommand(prime),
		newTestCommand(prime),
		newCommitCommand(prime),
		newPublish(prime),
		newEvalCommand(prime),
		artifactsCmd,
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
	EnvironmentSetupGroup = captain.NewCommandGroup(locale.Tl("group_environment_setup", "Environment Setup"), 10)
	EnvironmentUsageGroup = captain.NewCommandGroup(locale.Tl("group_environment_usage", "Environment Usage"), 9)
	ProjectUsageGroup     = captain.NewCommandGroup(locale.Tl("group_project_usages", "Project Usage"), 8)
	PackagesGroup         = captain.NewCommandGroup(locale.Tl("group_packages", "Package Management"), 7)
	PlatformGroup         = captain.NewCommandGroup(locale.Tl("group_tools", "Platform"), 6)
	VCSGroup              = captain.NewCommandGroup(locale.Tl("group_vcs", "Version Control"), 5)
	AutomationGroup       = captain.NewCommandGroup(locale.Tl("group_automation", "Automation"), 4)
	UtilsGroup            = captain.NewCommandGroup(locale.Tl("group_utils", "Utilities"), 3)
	AuthorGroup           = captain.NewCommandGroup(locale.Tl("group_author", "Author"), 6)
)

func newGlobalOptions() *globalOptions {
	return &globalOptions{}
}

func newStateCommand(globals *globalOptions, prime *primer.Values) *captain.Command {
	opts := state.NewOptions()
	var help bool

	runner := state.New(opts, prime)
	cmd := captain.NewCommand(
		"state",
		"",
		locale.T("state_description"),
		prime,
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
					if !condition.InUnitTest() {
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
				Name:        "non-interactive", // Name and Shorthand should be kept in sync with cmd/state/output.go
				Description: locale.T("flag_state_non_interactive_description"),
				Shorthand:   "n",
				Persist:     true,
				Value:       &globals.NonInteractive,
			},
			{
				Name:        "version",
				Description: locale.T("flag_state_version_description"),
				Value:       &opts.Version,
			},
			{
				Name:        "help",
				Description: locale.Tl("flag_help", "Help for this command"),
				Shorthand:   "h",
				Persist:     true,
				Value:       &help, // need to store the value somewhere, but Cobra handles this flag by itself
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

	cmd.SetHasVariableArguments()
	cmd.OnExecStart(cmdCall.OnExecStart)
	cmd.OnExecStop(cmdCall.OnExecStop)
	cmd.SetSupportsStructuredOutput()

	return cmd
}

// Execute runs the CmdTree using the provided CLI arguments.
func (ct *CmdTree) Execute(args []string) error {
	defer profile.Measure("cmdtree:Execute", time.Now())
	return ct.cmd.Execute(args)
}

func (ct *CmdTree) OnExecStart(handler captain.ExecEventHandler) {
	ct.cmd.OnExecStart(handler)
}

func (ct *CmdTree) OnExecStop(handler captain.ExecEventHandler) {
	ct.cmd.OnExecStop(handler)
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
		a.prime,
		aliased.Flags(),
		aliased.Arguments(),
		func(c *captain.Command, args []string) error {
			msg := locale.Tl(
				"cmd_deprecated_notice",
				"This command is deprecated. Please use '[ACTIONABLE]state {{.V0}}[/RESET]' instead.",
				aliased.Name(),
			)

			a.prime.Output().Notice(msg)

			return aliased.ExecuteFunc()(c, args)
		},
	)

	cmd.SetHidden(true)
	cmd.SetHasVariableArguments()

	a.parent.AddChildren(cmd)
}
