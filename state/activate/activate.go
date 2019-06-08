package activate

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
)

var (
	failInvalidNamespace = failures.Type("activate.fail.invalidnamespace", failures.FailUserInput)
	failTargetDirExists  = failures.Type("activate.fail.direxists", failures.FailUserInput)
)

// NamespaceRegex matches the org and project name in a namespace, eg. ORG/PROJECT
const NamespaceRegex = `^([\w-_]+)\/([\w-_\.]+)$`

var prompter prompt.Prompter

func init() {
	prompter = prompt.New()
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
	Path string
}

// Args hold the arg values passed through the command line
var Args struct {
	Namespace string
}

// Execute the activate command
func Execute(cmd *cobra.Command, args []string) {
	updater.PrintUpdateMessage()
	fail := auth.RequireAuthentication(locale.T("auth_required_activate"))
	if fail != nil {
		failures.Handle(fail, locale.T("err_activate_auth_required"))
	}

	var wg sync.WaitGroup

	logging.Debug("Execute")
	if Args.Namespace != "" {
		fail := activateFromNamespace(Args.Namespace)
		if fail != nil {
			failures.Handle(fail, locale.T("err_activate_namespace"))
			return
		}
	}

	project := project.Get()
	print.Info(locale.T("info_activating_state", project))
	venv := virtualenvironment.Get()
	venv.OnDownloadArtifacts(func() { print.Line(locale.T("downloading_artifacts")) })
	venv.OnInstallArtifacts(func() { print.Line(locale.T("installing_artifacts")) })
	fail = venv.Activate()
	if fail != nil {
		failures.Handle(fail, locale.T("error_could_not_activate_venv"))
		return
	}

	// Save path to project for future use
	savePathForNamespace(fmt.Sprintf("%s/%s", project.Owner(), project.Name()), filepath.Dir(project.Source().Path()))

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

// activateFromNamespace will try to find a relevant local checkout for the given namespace, or otherwise prompt the user
// to create one. Once that is done it changes directory to the checkout and defers activation back to the main execution handler.
func activateFromNamespace(namespace string) *failures.Failure {
	rx := regexp.MustCompile(NamespaceRegex)
	groups := rx.FindStringSubmatch(namespace)
	if len(groups) != 3 {
		return failInvalidNamespace.New(locale.Tr("err_invalid_namespace", namespace))
	}

	org := groups[1]
	name := groups[2]

	// Ensure that the project exists and that we have access to it
	project, fail := model.FetchProjectByName(org, name)
	if fail != nil {
		return fail
	}

	branch, fail := model.DefaultBranchForProject(project)
	if fail != nil {
		return fail
	}

	languages, fail := model.FetchLanguagesForBranch(branch)
	if fail != nil {
		return fail
	}

	var directory string

	// Change to already checked out project if it exists
	projectPaths := getPathsForNamespace(namespace)
	if len(projectPaths) > 0 {
		confirmedPath, fail := confirmProjectPath(projectPaths)
		if fail != nil {
			return fail
		}
		if confirmedPath != nil {
			directory = *confirmedPath
		}
	}

	// Otherwise ask the user for the directory
	if directory == "" {
		// Determine where to create our project
		directory, fail = determineProjectPath(namespace)
		if fail != nil {
			return fail
		}

		// Actually create the project
		fail = createProject(org, name, languages, directory)
		if fail != nil {
			return fail
		}
	}

	err := os.Chdir(directory)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	return nil
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

// createProject will create a project file (activestate.yaml) at the given location
func createProject(org, project string, languages []string, directory string) *failures.Failure {
	err := os.MkdirAll(directory, 0755)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	projectURL := fmt.Sprintf("https://%s/%s/%s/", constants.PlatformURL, org, project)
	pj := projectfile.Project{
		Project:   projectURL,
		Languages: []projectfile.Language{},
	}

	for _, language := range languages {
		pj.Languages = append(pj.Languages, projectfile.Language{Name: language})
	}

	pj.SetPath(filepath.Join(directory, constants.ConfigFileName))
	fail := pj.Save()
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

	if fileutils.DirExists(directory) {
		return "", failTargetDirExists.New(locale.Tr("err_namespace_dir_exists"))
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
