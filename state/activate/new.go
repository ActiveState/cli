package activate

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/organizations"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var exit = os.Exit

type projectStruct struct {
	name,
	owner,
	path,
	project string
}

// NewExecute creates a new project on the platform
func NewExecute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")
	proj := projectCreatePrompts()

	// Create the project locally on disk.
	if _, fail := projectfile.Create(proj.project, proj.path); fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_aborted"))
		exit(1)
	}

	print.Line(locale.T("state_activate_new_created", map[string]interface{}{"Dir": proj.path}))
}

// CopyExecute creates a new project from an existing activestate.yaml
func CopyExecute(cmd *cobra.Command, args []string) {
	projFile := project.Get().Source()
	projFile.Project = projectCreatePrompts().project
	projFile.Save()
}

func projectCreatePrompts() projectStruct {
	var defaultName string
	if projectExists(Flags.Path) {
		proj := project.Get()
		defaultName = proj.Name()
	}

	name, fail := prompter.Input(locale.T("state_activate_new_prompt_name"), defaultName, prompt.InputRequired)
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_aborted"))
		exit(1)
	}

	lang, fail := promptForLanguage()
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_aborted"))
		exit(1)
	}

	if !authentication.Get().Authenticated() && !condition.InTest() {
		print.Error(locale.T("error_state_activate_new_no_auth"))
		exit(1)
	}

	// If the user is not yet authenticated into the ActiveState Platform, it is a
	// simple prompt. Otherwise, fetch the list of organizations the user belongs
	// to and present the list to the user for a selection.
	owner, fail := promptForOwner()
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_aborted"))
		exit(1)
	}

	// Create the project on the platform
	if fail = createPlatformProject(name, owner, lang); fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_project_add"))
		exit(1)
	}

	path := Flags.Path
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			failures.Handle(err, locale.T("error_state_activate_new_aborted"))
			exit(1)
		}
	}

	// Create the project directory
	if fail := createProjectDir(path); fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_aborted"))
		exit(1)
	}

	cid, fail := model.LatestCommitID(owner, name)
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_no_commit_aborted",
			map[string]interface{}{"Owner": owner, "ProjectName": name}))

		exit(1)
	}

	projectURL := fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, owner, name)
	if cid == nil || cid.String() == "" {
		print.Warning(locale.T("error_state_activate_new_no_commit_aborted",
			map[string]interface{}{"Owner": owner, "ProjectName": name}))
	} else {
		projectURL = projectURL + fmt.Sprintf("?commitID=%s", cid.String())
	}
	return projectStruct{name: name, owner: owner, path: path, project: projectURL}
}

func promptForLanguage() (language.Language, *failures.Failure) {
	langs := language.Available()

	var ls []string
	for _, l := range langs {
		ls = append(ls, l.Text())
	}
	ls = append(ls, locale.T("state_activate_new_language_none"))

	if len(ls) > 1 {
		sel, fail := prompter.Select(locale.T("state_activate_new_prompt_language"), ls, "")
		if fail != nil {
			return language.Unknown, fail
		}

		for _, l := range langs {
			if l.Text() == sel {
				return l, nil
			}
		}
	}

	return language.Unknown, nil
}

func promptForOwner() (string, *failures.Failure) {
	params := organizations.NewListOrganizationsParams()
	memberOnly := true
	params.SetMemberOnly(&memberOnly)
	orgs, err := authentication.Client().Organizations.ListOrganizations(params, authentication.ClientAuth())
	if err != nil {
		return "", api.FailUnknown.New("error_state_activate_new_fetch_organizations")
	}
	owners := []string{}
	for _, org := range orgs.Payload {
		owners = append(owners, org.Name)
	}
	if len(owners) > 1 {
		return prompter.Select(locale.T("state_activate_new_prompt_owner"), owners, "")
	}
	return owners[0], nil // auto-select only option
}

func createPlatformProject(name, owner string, lang language.Language) *failures.Failure {
	addParams := projects.NewAddProjectParams()
	addParams.SetOrganizationName(owner)
	addParams.SetProject(&mono_models.Project{Name: name})
	_, err := authentication.Client().Projects.AddProject(addParams, authentication.ClientAuth())
	if err != nil {
		return api.FailUnknown.New(api.ErrorMessageFromPayload(err))
	}

	return model.CommitInitial(owner, name, lang.Requirement(), lang.RecommendedVersion())
}

func createProjectDir(path string) *failures.Failure {
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return failures.FailOS.Wrap(err)
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			return failures.FailOS.Wrap(err)
		}
	}
	return nil
}
