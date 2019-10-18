package activate

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/go-openapi/strfmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hail"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/cmdlets/checker"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var (
	failInvalidNamespace = failures.Type("activate.fail.invalidnamespace", failures.FailUserInput)
	failTargetDirInUse   = failures.Type("activate.fail.dirinuse", failures.FailUserInput)
)

var branchName = constants.BranchName

var (
	prompter prompt.Prompter
	repo     git.Repository
)

func init() {
	prompter = prompt.New()
	repo = git.NewRepo()
}

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
			Name:        "new",
			Shorthand:   "",
			Description: "flag_state_activate_new_description",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.New,
		},
		&commands.Flag{
			Name:        "owner",
			Shorthand:   "",
			Description: "flag_state_activate_owner_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Owner,
		},
		&commands.Flag{
			Name:        "project",
			Shorthand:   "",
			Description: "flag_state_activate_project_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Project,
		},
		&commands.Flag{
			Name:        "language",
			Shorthand:   "",
			Description: "flag_state_activate_language_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Language,
		},
	},
	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "arg_state_activate_namespace",
			Description: "arg_state_activate_namespace_description",
			Variable:    &Args.Namespace,
		},
	},
}

// Flags hold the flag values passed through the command line
var Flags struct {
	Path     string
	New      bool
	Owner    string
	Project  string
	Language string
}

// Args hold the arg values passed through the command line
var Args struct {
	Namespace string
}

// Execute the activate command
func Execute(cmd *cobra.Command, args []string) {
	updater.PrintUpdateMessage()

	switch {
	case len(args) == 0 && !projectExists(Flags.Path), Flags.New:
		NewExecute(cmd, args)
	default:
		ExistingExecute(cmd, args)
	}
}

func projectExists(path string) bool {
	prj := getProjectFileByPath(path)
	if prj == nil {
		return false
	}
	return true
}

// ExistingExecute activates a project based on the namespace in the
// arguments or the existing project file
func ExistingExecute(cmd *cobra.Command, args []string) {
	checker.RunCommitsBehindNotifier()

	if Args.Namespace != "" {
		fail := activateFromNamespace(Args.Namespace)
		if fail != nil {
			failures.Handle(fail, locale.T("err_activate_namespace"))
			return
		}
	}

	fail := promptCreateProjectIfNecessary(cmd, args)
	if fail != nil {
		failures.Handle(fail, locale.T("err_activate_create_project"))
		return
	}

	activateProject()
}

// activateFromNamespace will try to find a relevant local checkout for the given namespace, or otherwise prompt the user
// to create one. Once that is done it changes directory to the checkout and defers activation back to the main execution handler.
func activateFromNamespace(namespace string) *failures.Failure {
	ns, fail := project.ParseNamespace(namespace)
	if fail != nil {
		return fail
	}

	// Ensure that the project exists and that we have access to it
	project, fail := model.FetchProjectByName(ns.Owner, ns.Project)
	if fail != nil && fail.Type.Matches(model.FailNoValidProject) && !authentication.Get().Authenticated() {
		// If we can't find the project and we aren't authenticated we assume authentication is required
		fail = auth.RequireAuthentication(locale.T("auth_required_activate"))
		if fail != nil {
			return fail
		}
		return activateFromNamespace(namespace)
	} else if fail != nil {
		return fail
	}

	branch, fail := model.DefaultBranchForProject(project)
	if fail != nil {
		return fail
	}

	var directory string
	directory, fail = getDirByNameSpace(Flags.Path, namespace)
	if fail != nil {
		return fail
	}

	if _, err := os.Stat(filepath.Join(directory, constants.ConfigFileName)); err != nil {
		if project.RepoURL != nil {
			fail = cloneProjectRepo(ns.Owner, ns.Project, directory, branch.CommitID)
			if fail != nil {
				return fail
			}
		} else {
			fail = createProjectFile(ns.Owner, ns.Project, directory, branch.CommitID)
			if fail != nil {
				return fail
			}
		}
	} else {
		prj := getProjectFileByPath(directory)
		if !strings.Contains(prj.URL(), namespace) {
			return failTargetDirInUse.New(locale.Tr("err_namespace_and_project_do_not_match"))
		}
	}

	projectfile.Reset()
	err := os.Chdir(directory)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	return nil
}

func getDirByNameSpace(path string, namespace string) (string, *failures.Failure) {
	if Flags.Path != "" {
		return Flags.Path, nil
	}
	// Change to already checked out project if it exists
	projectPaths := getPathsForNamespace(namespace)
	if len(projectPaths) > 0 {
		confirmedPath, fail := confirmProjectPath(projectPaths)
		if fail != nil {
			return "", fail
		}
		if confirmedPath != nil {
			return *confirmedPath, nil
		}
	}
	return determineProjectPath(namespace)
}

func cloneProjectRepo(org, name, directory string, commitID *strfmt.UUID) *failures.Failure {
	fail := repo.CloneProject(org, name, directory)
	if fail != nil {
		return fail
	}
	_, err := os.Stat(filepath.Join(directory, constants.ConfigFileName))
	if os.IsNotExist(err) {
		fail = createProjectFile(org, name, directory, commitID)
		if fail != nil {
			return fail
		}
	} else if err != nil {
		return failures.FailOS.Wrap(err)
	}

	return nil
}

func getProjectFileByPath(path string) *project.Project {
	if path != "" {
		// CWD is used to return to the directory before retrieving the as.yaml
		// file was initiated.
		cwd, err := os.Getwd()
		if err != nil {
			failures.Handle(err, locale.Tr("err_activate_path", path))
		}

		if err := os.Chdir(path); err != nil {
			failures.Handle(err, locale.Tr("err_activate_path", path))
		}
		defer func() {
			logging.Debug("moving back to origin dir")
			if err := os.Chdir(cwd); err != nil {
				failures.Handle(err, locale.Tr("err_activate_path", path))
			}
		}()
	}

	prj, fail := project.GetOnce()
	if fail != nil {
		if fileutils.FailFindInPathNotFound.Matches(fail.Type) {
			return nil
		}
	}
	return prj
}

func activateProject() {
	// activate should be continually called while returning true
	// looping here provides a layer of scope to handle printing output
	var proj *project.Project
	for {
		proj = project.Get()
		print.Info(locale.T("info_activating_state", proj))

		if branchName != constants.StableBranch {
			print.Stderr().Warning(locale.Tr("unstable_version_warning", constants.BugTrackerURL))
		}

		if !activate(proj.Owner(), proj.Name(), proj.Source().Path()) {
			break
		}

		print.Info(locale.T("info_reactivating", proj))
	}

	print.Bold(locale.T("info_deactivated", proj))
}

// savePathForNamespace saves a new path for the given namespace, so the state tool is aware of locations where this
// namespace is used
func savePathForNamespace(namespace, path string) {
	key := fmt.Sprintf("project_%s", namespace)
	paths := getPathsForNamespace(namespace)
	paths = append(paths, path)
	viper.Set(key, paths)
}

// getPathsForNamespace returns any locations that this namespace is used, it strips out duplicates and paths that are
// no longer valid
func getPathsForNamespace(namespace string) []string {
	key := fmt.Sprintf("project_%s", namespace)
	paths := viper.GetStringSlice(key)
	paths = funk.FilterString(paths, func(path string) bool {
		return fileutils.FileExists(filepath.Join(path, constants.ConfigFileName))
	})
	paths = funk.UniqString(paths)
	viper.Set(key, paths)
	return paths
}

func createProjectFile(org, project, directory string, commitID *strfmt.UUID) *failures.Failure {
	fail := fileutils.MkdirUnlessExists(directory)
	if fail != nil {
		return fail
	}

	projectURL := fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, org, project)
	if commitID != nil {
		projectURL = fmt.Sprintf("%s?commitID=%s", projectURL, commitID)
	}

	_, fail = projectfile.Create(projectURL, directory)
	if fail != nil {
		return fail
	}

	return nil
}

// determineProjectPath will prompt the user for a location to save the project at
func determineProjectPath(namespace string) (string, *failures.Failure) {
	wd, err := os.Getwd()
	if err != nil {
		return "", failures.FailRuntime.Wrap(err)
	}

	directory, fail := prompter.Input(locale.Tr("activate_namespace_location", namespace), filepath.Join(wd, namespace))
	if fail != nil {
		return "", fail
	}
	logging.Debug("Using: %s", directory)

	if fileutils.FileExists(filepath.Join(directory, constants.ConfigFileName)) {
		return "", failTargetDirInUse.New(locale.Tr("err_namespace_dir_inuse"))
	}

	return directory, nil
}

// confirmProjectPath will prompt the user for which project path they wish to use
func confirmProjectPath(projectPaths []string) (confirmedPath *string, fail *failures.Failure) {
	if len(projectPaths) == 0 {
		return nil, nil
	}

	noneStr := locale.T("activate_select_optout")
	choices := append(projectPaths, noneStr)
	path, fail := prompter.Select(locale.T("activate_namespace_existing"), choices, "")
	if fail != nil {
		return nil, fail
	}
	if path != "" && path != noneStr {
		return &path, nil
	}

	return nil, nil
}

func promptCreateProjectIfNecessary(cmd *cobra.Command, args []string) *failures.Failure {
	proj := project.Get()
	_, fail := model.FetchProjectByName(proj.Owner(), proj.Name())
	if fail == nil || !fail.Type.Matches(model.FailNoValidProject) {
		return fail
	}

	// If we can't find the project and we aren't authenticated we should first authenticate before continuing
	if !authentication.Get().Authenticated() {
		fail := auth.RequireAuthentication(locale.T("auth_required_activate"))
		if fail != nil {
			return fail
		}
		return promptCreateProjectIfNecessary(cmd, args)
	}

	if api.FailProjectNotFound.Matches(fail.Type) || model.FailNoValidProject.Matches(fail.Type) {
		create, fail := prompter.Confirm(locale.Tr("state_activate_prompt_create_project", proj.Name(), proj.Owner()), false)
		if fail != nil {
			return fail
		}
		if create {
			CopyExecute(cmd, args)
		} else {
			return failures.FailUserInput.New(locale.T("err_must_create_project"))
		}
	} else {
		return fail
	}

	return nil
}

// activate will activate the venv and subshell. It is meant to be run in a loop
// with the return value indicating whether another iteration is warranted.
func activate(owner, name, srcPath string) bool {
	// Ensure that the project exists and that we have access to it
	_, fail := model.FetchProjectByName(owner, name)
	if fail != nil && fail.Type.Matches(model.FailNoValidProject) {
		// If we can't find the project and we aren't authenticated we assume authentication is required
		fail = auth.RequireAuthentication(locale.T("auth_required_activate"))
		if fail != nil {
			failures.Handle(fail, locale.T("err_activate_auth_required"))
			return false
		}
	}

	venv := virtualenvironment.Get()
	venv.OnDownloadArtifacts(func() { print.Line(locale.T("downloading_artifacts")) })
	venv.OnInstallArtifacts(func() { print.Line(locale.T("installing_artifacts")) })
	fail = venv.Activate()
	if fail != nil {
		failures.Handle(fail, locale.T("error_could_not_activate_venv"))
		return false
	}

	// Save path to project for future use
	savePathForNamespace(fmt.Sprintf("%s/%s", owner, name), filepath.Dir(srcPath))

	ignoreWindowsInterrupts()

	subs, err := subshell.Activate()
	if err != nil {
		failures.Handle(err, locale.T("error_could_not_activate_subshell"))
		return false
	}

	if condition.InTest() {
		return false
	}

	done := make(chan struct{})
	defer close(done)
	fname := path.Join(config.ConfigPath(), constants.UpdateHailFileName)

	hails, fail := hail.Open(done, fname)
	if fail != nil {
		failures.Handle(fail, locale.T("error_unable_to_monitor_pulls"))
		return false
	}

	return listenForReactivation(venv.ActivationID(), hails, subs)
}

func ignoreWindowsInterrupts() {
	if runtime.GOOS == "windows" {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT)
		go func() {
			for range c {
			}
		}()
	}
}

type subShell interface {
	Deactivate() *failures.Failure
	Failures() <-chan *failures.Failure
}

func listenForReactivation(id string, rcvs <-chan *hail.Received, subs subShell) bool {
	for {
		select {
		case rcvd, ok := <-rcvs:
			if !ok {
				logging.Error("hailing channel closed")
				return false
			}

			if rcvd.Fail != nil {
				logging.Error("error in hailing channel: %s", rcvd.Fail)
				continue
			}

			if !idsValid(id, rcvd.Data) {
				continue
			}

			// A subshell will have triggered this case; Wait for
			// output completion before deactivating. The nature of
			// this issue is unclear at this time.
			time.Sleep(time.Second)

			if fail := subs.Deactivate(); fail != nil {
				failures.Handle(fail, locale.T("error_deactivating_subshell"))
				return false
			}

			return true

		case fail, ok := <-subs.Failures():
			if !ok {
				logging.Error("subshell failure channel closed")
				return false
			}

			if fail != nil {
				failures.Handle(fail, locale.T("error_in_active_subshell"))
			}

			return false
		}
	}
}

func idsValid(currID string, rcvdID []byte) bool {
	return currID != "" && len(rcvdID) > 0 && currID == string(rcvdID)
}
