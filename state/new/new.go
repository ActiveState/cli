package new

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/organizations"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var exit = os.Exit

// Command is the new command's definition.
var Command = &commands.Command{
	Name:        "new",
	Description: "new_project",
	Run:         Execute,

	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "path",
			Shorthand:   "p",
			Description: "flag_state_new_path_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Path,
		},
		&commands.Flag{
			Name:        "owner",
			Shorthand:   "o",
			Description: "flag_state_new_owner_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Owner,
		},
	},

	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "arg_state_new_name",
			Description: "arg_state_new_name_description",
			Variable:    &Args.Name,
		},
	},
}

// Flags hold the flag values passed through the command line.
var Flags struct {
	Path  string
	Owner string
}

// Args hold the arg values passed through the command line.
var Args struct {
	Name string
}

var prompter prompt.Prompter

func init() {
	prompter = prompt.New()
}

// Execute the new command.
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")

	if !authentication.Get().Authenticated() && flag.Lookup("test.v") == nil {
		print.Error(locale.T("error_state_new_no_auth"))
		exit(1)
	}

	// If project name was not given, ask for it.
	if Args.Name == "" {
		var fail *failures.Failure
		Args.Name, fail = prompter.Input(locale.T("state_new_prompt_name"), "", prompt.InputRequired)
		if fail != nil {
			failures.Handle(fail, locale.T("error_state_new_aborted"))
			exit(1)
		}
	}

	// If owner argument was not given, ask for it.
	// If the user is not yet authenticated into the ActiveState Platform, it is a
	// simple prompt. Otherwise, fetch the list of organizations the user belongs
	// to and present the list to the user for a selection.
	if Flags.Owner == "" {
		var fail *failures.Failure
		Flags.Owner, fail = promptForOwner()
		if fail != nil {
			failures.Handle(fail, locale.T("error_state_new_aborted"))
			exit(1)
		}
	}

	// Create the project on the platform
	fail := createPlatformProject()
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_new_project_add"))
		exit(1)
	}

	// If path argument was not given, infer it from the current working directory
	// and the project name given.
	// Otherwise, ensure the given path does not already exist.
	if Flags.Path == "" {
		var fail *failures.Failure
		Flags.Path, fail = fetchPath()
		if fail != nil {
			failures.Handle(fail, locale.T("error_state_new_aborted"))
			exit(1)
		}
	}

	// Create the project directory
	fail = createProjectDir()
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_new_aborted"))
		exit(1)
	}

	// Create the project locally on disk.
	project := projectfile.Project{
		Name:  Args.Name,
		Owner: Flags.Owner,
	}
	project.SetPath(filepath.Join(Flags.Path, constants.ConfigFileName))
	project.Save()
	print.Line(locale.T("state_new_created", map[string]interface{}{"Dir": Flags.Path}))
}

func promptForOwner() (string, *failures.Failure) {
	params := organizations.NewListOrganizationsParams()
	memberOnly := true
	params.SetMemberOnly(&memberOnly)
	orgs, err := authentication.Client().Organizations.ListOrganizations(params, authentication.ClientAuth())
	if err != nil {
		return "", api.FailUnknown.New("error_state_new_fetch_organizations")
	}
	owners := []string{}
	for _, org := range orgs.Payload {
		owners = append(owners, org.Name)
	}
	if len(owners) > 1 {
		return prompter.Select(locale.T("state_new_prompt_owner"), owners, Flags.Owner)
	}
	return owners[0], nil // auto-select only option
}

func fetchPath() (string, *failures.Failure) {
	cwd, _ := os.Getwd()
	files, _ := ioutil.ReadDir(cwd)

	if len(files) == 0 {
		// Current working directory is devoid of files. Use it as the path for
		// the new project.
		return cwd, nil
	}

	// Current working directory has files in it. Use a subdirectory with the
	// project name as the path for the new project.
	path := filepath.Join(cwd, Args.Name)
	if _, err := os.Stat(path); err == nil {
		return "", failures.FailIO.New("error_state_new_exists")
	}

	return path, nil
}

func createPlatformProject() *failures.Failure {
	addParams := projects.NewAddProjectParams()
	addParams.SetOrganizationName(Flags.Owner)
	addParams.SetProject(&mono_models.Project{Name: Args.Name})
	_, err := authentication.Client().Projects.AddProject(addParams, authentication.ClientAuth())
	if err != nil {
		return api.FailUnknown.Wrap(err)
	}
	return nil
}

func createProjectDir() *failures.Failure {
	if _, err := os.Stat(Flags.Path); err == nil {
		// Directory already exists
		files, _ := ioutil.ReadDir(Flags.Path)
		if len(files) == 0 {
			return nil
		}
		return failures.FailIO.New("error_state_new_exists")
	}
	if err := os.MkdirAll(Flags.Path, 0755); err != nil {
		return failures.FailIO.New("error_state_new_mkdir")
	}
	return nil
}
