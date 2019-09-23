package activate

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

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
	path, fail := fetchPath(proj.name)
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_aborted"))
		exit(1)
	}

	// Create the project locally on disk.
	if _, fail = projectfile.Create(proj.project, proj.path); fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_aborted"))
		exit(1)
	}

	print.Line(locale.T("state_activate_new_created", map[string]interface{}{"Dir": path}))
}

// CopyExecute creates a new project from an existing activestate.yaml
func CopyExecute(cmd *cobra.Command, args []string) {
	projFile := project.Get().Source()
	projFile.Project = projectCreatePrompts().project
	projFile.Save()
}

func projectCreatePrompts() projectStruct {
	var defaultName string
	if projectExists() {
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

	if !authentication.Get().Authenticated() && flag.Lookup("test.v") == nil {
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

	path, fail := fetchPath(name)
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_aborted"))
		exit(1)
	}

	// Create the project directory
	if fail := createProjectDir(path); fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_aborted"))
		exit(1)
	}

	var commitID string
	cid, fail := model.LatestCommitID(owner, name)
	if fail != nil {
		failures.Handle(fail, locale.T("error_state_activate_new_no_commit_aborted",
			map[string]interface{}{"Owner": owner, "ProjectName": name}))

		exit(1)
	}

	if cid != nil {
		commitID = cid.String()
	}

	projectURL := fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, owner, name)
	if commitID == "" {
		print.Warning(locale.T("error_state_activate_new_no_commit_aborted",
			map[string]interface{}{"Owner": owner, "ProjectName": name}))
	} else {
		projectURL = projectURL + fmt.Sprintf("?commitID=%s", commitID)
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

func fetchPath(projName string) (string, *failures.Failure) {
	cwd, _ := os.Getwd()
	files, _ := ioutil.ReadDir(cwd)

	if len(files) == 0 {
		// Current working directory is devoid of files. Use it as the path for
		// the new project.
		return cwd, nil
	}

	// Current working directory has files in it. Use a subdirectory with the
	// project name as the path for the new project.
	path := filepath.Join(cwd, projName)
	if _, err := os.Stat(path); err == nil {
		return "", failures.FailIO.New("error_state_activate_new_exists")
	}

	return path, nil
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
	if _, err := os.Stat(path); err == nil {
		// Directory already exists
		files, _ := ioutil.ReadDir(path)
		if len(files) == 0 {
			return nil
		}
		return failures.FailIO.New("error_state_activate_new_exists")
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return failures.FailIO.New("error_state_activate_new_mkdir")
	}
	return nil
}
