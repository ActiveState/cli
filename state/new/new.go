package new

import (
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ActiveState/cli/pkg/platform/authentication"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/organizations"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/spf13/cobra"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

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
		&commands.Flag{
			Name:        "version",
			Shorthand:   "v",
			Description: "flag_state_new_version_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Version,
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
	Path    string
	Owner   string
	Version string
}

// Args hold the arg values passed through the command line.
var Args struct {
	Name string
}

// Execute the new command.
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")

	if !authentication.Get().Authenticated() && flag.Lookup("test.v") == nil {
		print.Error(locale.T("error_state_new_no_auth"))
		return
	}

	// If project name was not given, ask for it.
	if Args.Name == "" {
		prompter := &survey.Input{Message: locale.T("state_new_prompt_name")}
		if err := survey.AskOne(prompter, &Args.Name, prompt.ValidateRequired); err != nil {
			print.Error(locale.T("error_state_new_aborted"))
			return
		}
	}

	// If owner argument was not given, ask for it.
	// If the user is not yet authenticated into the ActiveState Platform, it is a
	// simple prompt. Otherwise, fetch the list of organizations the user belongs
	// to and present the list to the user for a selection.
	if Flags.Owner == "" {
		params := organizations.NewListOrganizationsParams()
		memberOnly := true
		params.SetMemberOnly(&memberOnly)
		orgs, err := authentication.Client().Organizations.ListOrganizations(params, authentication.ClientAuth())
		if err != nil {
			logging.Errorf("Unable to fetch organizations: %s", err)
			print.Error(locale.T("error_state_new_fetch_organizations"))
			return
		}
		owners := []string{}
		for _, org := range orgs.Payload {
			owners = append(owners, org.Name)
		}
		if len(owners) > 1 {
			prompt := &survey.Select{
				Message: locale.T("state_new_prompt_owner"),
				Options: owners,
			}
			if err = survey.AskOne(prompt, &Flags.Owner, nil); err != nil {
				print.Error(locale.T("error_state_new_aborted"))
				return
			}
		} else {
			Flags.Owner = owners[0] // auto-select only option
		}
	}

	// Create the project on the ActiveState Platform.
	addParams := projects.NewAddProjectParams()
	addParams.SetOrganizationName(Flags.Owner)
	addParams.SetProject(&mono_models.Project{Name: Args.Name})
	_, err := authentication.Client().Projects.AddProject(addParams, authentication.ClientAuth())
	if err != nil {
		logging.Errorf("Unable to create Platform project: %s", err)
		print.Error(locale.T("error_state_new_project_add"))
		return
	}

	// If path argument was not given, infer it from the current working directory
	// and the project name given.
	// Otherwise, ensure the given path does not already exist.
	if Flags.Path == "" {
		cwd, _ := os.Getwd()
		files, _ := ioutil.ReadDir(cwd)
		if len(files) == 0 {
			// Current working directory is devoid of files. Use it as the path for
			// the new project.
			Flags.Path = cwd
		} else {
			// Current working directory has files in it. Use a subdirectory with the
			// project name as the path for the new project.
			Flags.Path = filepath.Join(cwd, Args.Name)
			if _, err := os.Stat(Flags.Path); err == nil {
				print.Error(locale.T("error_state_new_exists"))
				return
			}
		}
	} else if _, err := os.Stat(Flags.Path); err == nil {
		print.Error(locale.T("error_state_new_exists"))
		return
	}
	if err := os.MkdirAll(Flags.Path, 0755); err != nil {
		logging.Errorf("Unable to create new project directory: %s", err)
		print.Error(locale.T("error_state_new_mkdir"))
		return
	}

	// If version argument was not given, ask for it.
	// Otherwise, validate its format.
	if Flags.Version == "" {
		prompt := &survey.Input{Message: locale.T("state_new_prompt_version")}
		err := survey.AskOne(prompt, &Flags.Version, func(val interface{}) error {
			if !regexp.MustCompile("^\\d+(\\.\\d+)*$").MatchString(val.(string)) {
				return errors.New(locale.T("error_state_new_prompt_version"))
			}
			return nil
		})
		if err != nil {
			print.Error(locale.T("error_state_new_aborted"))
			return
		}
	} else {
		if !regexp.MustCompile("^\\d+(\\.\\d+)*$").MatchString(Flags.Version) {
			print.Error(locale.T("error_state_new_version"))
			return
		}
	}

	// Create the project locally on disk.
	project := projectfile.Project{
		Name:    Args.Name,
		Owner:   Flags.Owner,
		Version: Flags.Version,
	}
	project.SetPath(filepath.Join(Flags.Path, constants.ConfigFileName))
	project.Save()
	print.Line(locale.T("state_new_created", map[string]interface{}{"Dir": Flags.Path}))
}
