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
	name     string
	owner    string
	language language.Language
}

// NewExecute creates a new project on the platform
func NewExecute(cmd *cobra.Command, args []string) {
	var (
		projectInfo *projectStruct
		fail        *failures.Failure
	)
	logging.Debug("Execute")

	projectInfo, fail = newProjectInfo()
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_prompt"))
		return
	}

	fail = createNewProject(projectInfo)
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_create"))
		return
	}

	activateProject()
}

// CopyExecute creates a new project from an existing activestate.yaml
func CopyExecute(cmd *cobra.Command, args []string) {
	projFile := project.Get().Source()

	projectInfo, fail := newProjectInfo()
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_copy_prompts"))
		return
	}

	projFile.Project, fail = getProjectURL(projectInfo.owner, projectInfo.name)
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_copy_project_url"))
		return
	}

	fail = projFile.Save()
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_copy_save"))
		return
	}
}

func newProjectInfo() (*projectStruct, *failures.Failure) {
	projectInfo := new(projectStruct)

	var fail *failures.Failure
	projectInfo.name = Flags.Project
	if projectInfo.name == "" {
		projectInfo.name, fail = promptForProjectName()
		if fail != nil {
			return nil, fail
		}
	}

	if Flags.Language == "" {
		projectInfo.language, fail = promptForLanguage()
		if fail != nil {
			return nil, fail
		}
	} else {
		projectInfo.language, fail = getLanguageFromFlags()
		if fail != nil {
			return nil, fail
		}
	}

	if !authentication.Get().Authenticated() && !condition.InTest() {
		return nil, failures.FailUser.New(locale.T("error_state_activate_new_no_auth"))
	}

	// If the user is not yet authenticated into the ActiveState Platform, it is a
	// simple prompt. Otherwise, fetch the list of organizations the user belongs
	// to and present the list to the user for a selection.
	projectInfo.owner = Flags.Owner
	if projectInfo.owner == "" {
		projectInfo.owner, fail = promptForOwner()
		if fail != nil {
			return nil, fail
		}
	}

	return projectInfo, nil
}

func promptForProjectName() (string, *failures.Failure) {
	var defaultName string
	if projectExists(Flags.Path) {
		proj := project.Get()
		defaultName = proj.Name()
	}

	return prompter.Input(locale.T("state_activate_new_prompt_name"), defaultName, prompt.InputRequired)
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

func createNewProject(projectInfo *projectStruct) *failures.Failure {
	if fail := createPlatformProject(projectInfo.name, projectInfo.owner, projectInfo.language); fail != nil {
		return fail
	}

	path, fail := getProjectPath()
	if fail != nil {
		return fail
	}

	if fail := createProjectDir(path); fail != nil {
		return fail
	}

	projectURL, fail := getProjectURL(projectInfo.owner, projectInfo.name)
	if fail != nil {
		return fail
	}

	if _, fail := projectfile.Create(projectURL, path); fail != nil {
		return fail
	}

	print.Line(locale.T("state_activate_new_created", map[string]interface{}{"Dir": path}))
	return nil
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

func getProjectPath() (string, *failures.Failure) {
	path := Flags.Path
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return "", failures.FailOS.Wrap(err)
		}
	}

	return path, nil
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

func getProjectURL(owner, name string) (string, *failures.Failure) {
	cid, fail := model.LatestCommitID(owner, name)
	if fail != nil {
		return "", fail
	}

	projectURL := fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, owner, name)
	if cid == nil || cid.String() == "" {
		print.Warning(locale.T("error_state_activate_new_no_commit_aborted",
			map[string]interface{}{"Owner": owner, "ProjectName": name}))
	} else {
		projectURL = projectURL + fmt.Sprintf("?commitID=%s", cid.String())
	}

	return projectURL, nil
}

func getLanguageFromFlags() (language.Language, *failures.Failure) {
	for _, lang := range language.Available() {
		if Flags.Language == lang.String() {
			return lang, nil
		}
	}

	return language.Unknown, failures.FailUserInput.New(locale.T("error_state_activate_language_flag_invalid"))
}
