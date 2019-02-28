package activate

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/ActiveState/cli/internal/projects"

	"github.com/thoas/go-funk"

	"github.com/spf13/viper"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cobra"
)

var (
	failInvalidNamespace = failures.Type("activate.fail.invalidnamespace", failures.FailUserInput)
	failTargetDirExists  = failures.Type("activate.fail.direxists", failures.FailUserInput)
)

const NamespaceRegex = `^([\w-_]+)\/([\w-_]+)$`

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

	var wg sync.WaitGroup

	logging.Debug("Execute")
	if Args.Namespace != "" {
		fail := activateFromNamespace(Args.Namespace)
		if fail != nil {
			failures.Handle(fail, locale.T("err_activate_namespace"))
			return
		}
	}

	project := projectfile.Get()
	print.Info(locale.T("info_activating_state", project))
	var fail = virtualenvironment.Activate()
	if fail != nil {
		failures.Handle(fail, locale.T("error_could_not_activate_venv"))
		return
	}

	// Save path to project for future use
	savePathForNamespace(fmt.Sprintf("%s/%s", project.Owner, project.Name), filepath.Dir(project.Path()))

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

func activateFromNamespace(namespace string) *failures.Failure {
	rx := regexp.MustCompile(NamespaceRegex)
	groups := rx.FindStringSubmatch(namespace)
	if len(groups) != 3 {
		return failInvalidNamespace.New(locale.Tr("err_invalid_namespace", namespace))
	}

	org := groups[1]
	name := groups[2]

	_, fail := projects.FetchByName(org, name)
	if fail != nil {
		return fail
	}

	// Change to already checked out project if it exists
	projectPaths := getPathsForNamespace(namespace)
	if len(projectPaths) > 0 {
		confirmedPath, fail := confirmProjectPath(projectPaths)
		if fail != nil {
			return fail
		}
		if confirmedPath != nil {
			os.Chdir(*confirmedPath)
			return nil
		}
	}

	// Determine where to create our project
	directory, fail := determineProjectPath(namespace)
	if fail != nil {
		return fail
	}

	// Actually create the project
	fail = createProject(org, name, directory)
	if fail != nil {
		return fail
	}

	os.Chdir(directory)
	return nil
}

func savePathForNamespace(namespace, path string) {
	key := fmt.Sprintf("project_%s", namespace)
	paths := getPathsForNamespace(namespace)
	paths = append(paths, path)
	viper.Set(key, paths)
}

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

func createProject(org, project, directory string) *failures.Failure {
	err := os.MkdirAll(directory, 0755)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	pj := projectfile.Project{
		Name:  project,
		Owner: org,
	}

	pj.SetPath(filepath.Join(directory, constants.ConfigFileName))
	pj.Save()

	return nil
}

func determineProjectPath(namespace string) (string, *failures.Failure) {
	wd, err := os.Getwd()
	if err != nil {
		return "", failures.FailRuntime.Wrap(err)
	}

	directory := filepath.Join(wd, namespace)
	survey.AskOne(&survey.Input{
		Message: locale.Tr("activate_namespace_location", namespace),
		Default: directory,
	}, &directory, nil)

	if fileutils.DirExists(directory) {
		return "", failTargetDirExists.New(locale.Tr("err_namespace_dir_exists"))
	}

	return directory, nil
}

func confirmProjectPath(projectPaths []string) (confirmedPath *string, fail *failures.Failure) {
	if len(projectPaths) == 0 {
		return nil, nil
	}

	var path string
	var noneStr = locale.T("activate_select_optout")
	err := survey.AskOne(&survey.Select{
		Message: locale.T("activate_namespace_existing"),
		Options: append(projectPaths, noneStr),
	}, &path, nil)
	if err != nil {
		return nil, failures.FailUserInput.Wrap(err)
	}
	if path != "" && path != noneStr {
		return &path, nil
	}

	return nil, nil
}
