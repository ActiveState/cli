package show

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/secrets"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/go-openapi/strfmt"
)

// Params describes the data required for the show run func.
type Params struct {
	Remote string
}

// Show manages the show run execution context.
type Show struct {
	project     *project.Project
	out         output.Outputer
	conditional *constraints.Conditional
	auth        *authentication.Auth
}

type auther interface {
	Authenticated() bool
}

type primeable interface {
	primer.Projecter
	primer.Outputer
	primer.Conditioner
	primer.Auther
}

type outputData struct {
	ProjectURL   string `locale:"project_url,Project URL"`
	Namespace    string
	Name         string
	Organization string
	Visibility   string `locale:"visibility,Visibility"`
	Commit       string `locale:"commit,Latest Commit"`
	Platforms    []string
	Languages    []string
	Secrets      *secretOutput     `locale:"secrets,Secrets"`
	Events       []string          `json:",omitempty"`
	Scripts      map[string]string `json:",omitempty"`
}

type secretOutput struct {
	User    []string `locale:"user,User"`
	Project []string `locale:"project,Project"`
}

// New returns a pointer to an instance of Show.
func New(prime primeable) *Show {
	return &Show{
		prime.Project(),
		prime.Output(),
		prime.Conditional(),
		prime.Auth(),
	}
}

// Run is the primary show logic.
func (s *Show) Run(params Params) error {
	logging.Debug("Execute show")

	var (
		owner       string
		projectName string
		projectURL  string
		events      []string
		scripts     map[string]string
		err         error
	)

	if params.Remote != "" {
		namespaced, fail := project.ParseNamespace(params.Remote)
		if fail != nil {
			return locale.WrapError(fail, "err_show_parse_namespace", "Invalid remote argument, must be of the form <Owner>/<Project>")
		}

		owner = namespaced.Owner
		projectName = namespaced.Project
	} else {
		if s.project == nil {
			return locale.NewError("err_no_projectfile")
		}

		owner = s.project.Owner()
		projectName = s.project.Name()
		projectURL = s.project.URL()

		events, err = eventsData(s.project.Source(), s.conditional)
		if err != nil {
			return locale.WrapError(err, "err_show_events", "Could not parse events")
		}

		scripts, err = scriptsData(s.project.Source(), s.conditional)
		if err != nil {
			return locale.WrapError(err, "err_show_scripts", "Could not parse scripts")
		}
	}

	remoteProject, fail := model.FetchProjectByName(owner, projectName)
	if fail != nil && fail.Type.Matches(model.FailProjectNotFound) {
		return locale.WrapError(fail, "err_show_project_not_found", "Please run `state push` to synchronize this project with the ActiveState Platform.")
	} else if fail != nil {
		return locale.WrapError(err, "err_show_get_project", "Could not get remote project details")
	}

	branch, fail := model.DefaultBranchForProjectName(owner, projectName)
	if fail != nil {
		return locale.WrapError(fail, "err_show_get_default_branch", "Could not get project information from the platform")
	}
	if branch.CommitID == nil {
		return locale.NewError("err_show_commitID", "Remote project details are incorrect. Default branch is missing commitID")
	}

	if projectURL == "" {
		projectURL = model.ProjectURL(owner, projectName, branch.CommitID.String())
	}

	platforms, err := platformsData(owner, projectName, *branch.CommitID)
	if err != nil {
		return locale.WrapError(err, "err_show_platforms", "Could not retrieve platform information")
	}

	languages, err := languagesData(owner, projectName)
	if err != nil {
		return locale.WrapError(err, "err_show_langauges", "Could not retrieve language information")
	}

	commit, err := commitsData(owner, projectName, *branch.CommitID, s.project, s.auth)
	if err != nil {
		return locale.WrapError(err, "err_show_commit", "Could not get commit information")
	}

	secrets, err := secretsData(owner, projectName, s.auth)
	if err != nil {
		return locale.WrapError(err, "err_show_secrets", "Could not get secret information")
	}

	data := outputData{
		ProjectURL:   projectURL,
		Namespace:    fmt.Sprintf("%s/%s", owner, projectName),
		Name:         projectName,
		Organization: owner,
		Visibility:   visibilityData(owner, projectName, remoteProject),
		Commit:       commit,
		Languages:    languages,
		Platforms:    platforms,
		Secrets:      secrets,
		Events:       events,
		Scripts:      scripts,
	}

	s.out.Print(data)
	return nil
}

func eventsData(project *projectfile.Project, conditional *constraints.Conditional) ([]string, error) {
	if len(project.Events) == 0 {
		return nil, nil
	}

	constrained, err := constraints.FilterUnconstrained(conditional, project.Events.AsConstrainedEntities())
	if err != nil {
		return nil, locale.WrapError(err, "err_event_condition", "Event has invalid conditional")
	}

	es := projectfile.MakeEventsFromConstrainedEntities(constrained)

	var data []string
	for _, event := range es {
		data = append(data, event.Name)
	}

	return data, nil
}

func scriptsData(project *projectfile.Project, conditional *constraints.Conditional) (map[string]string, error) {
	if len(project.Scripts) == 0 {
		return nil, nil
	}

	constrained, err := constraints.FilterUnconstrained(conditional, project.Scripts.AsConstrainedEntities())
	if err != nil {
		return nil, locale.WrapError(err, "err_script_condition", "Script has invalid conditional")
	}

	scripts := projectfile.MakeScriptsFromConstrainedEntities(constrained)

	data := make(map[string]string)
	for _, script := range scripts {
		data[script.Name] = script.Description
	}

	return data, nil
}

func platformsData(owner, project string, branchID strfmt.UUID) ([]string, error) {
	remotePlatforms, fail := model.FetchPlatformsForCommit(branchID)
	if fail != nil {
		return nil, locale.WrapError(fail, "err_show_get_platforms", "Could not get platform details for commit: {{.V0}}", branchID.String())
	}

	var platforms []string
	for _, plat := range remotePlatforms {
		if plat.DisplayName != nil {
			platforms = append(platforms, *plat.DisplayName)
		}
	}

	return platforms, nil
}

func languagesData(owner, project string) ([]string, error) {
	platformLanguages, fail := model.FetchLanguagesForProject(owner, project)
	if fail != nil {
		return nil, locale.WrapError(fail, "err_show_get_languages", "Could not get languages for project")
	}

	languages := make([]string, len(platformLanguages))
	for i, pl := range platformLanguages {
		languages[i] = fmt.Sprintf("%s-%s", pl.Name, pl.Version)
	}

	return languages, nil
}

func visibilityData(owner, project string, remoteProject *mono_models.Project) string {
	if remoteProject.Private {
		return locale.T("private")
	}
	return locale.T("public")
}

func commitsData(owner, project string, commitID strfmt.UUID, localProject *project.Project, auth auther) (string, error) {
	latestCommit, fail := model.LatestCommitID(owner, project)
	if fail != nil {
		return "", locale.WrapError(fail, "err_show_get_latest_commit", "Could not get latest commit ID")
	}

	if !auth.Authenticated() {
		return latestCommit.String(), nil
	}

	if localProject != nil && localProject.Owner() == owner && localProject.Name() == project {
		behind, fail := model.CommitsBehindLatest(owner, project, localProject.CommitID())
		if fail != nil {
			return "", locale.WrapError(fail, "err_show_commits_behind", "Could not determine number of commits behind latest")
		}
		if behind != 0 {
			return fmt.Sprintf("%s (%d behind latest)", localProject.CommitID(), behind), nil
		}
		return localProject.CommitID(), nil
	}

	return latestCommit.String(), nil
}

func secretsData(owner, project string, auth auther) (*secretOutput, error) {
	if !auth.Authenticated() {
		return nil, nil
	}

	client := secretsapi.Get()
	sec, fail := secrets.DefsByProject(client, owner, project)
	if fail != nil && fail.Type.Matches(api.FailAuth) {
		// The user is authenticated however may not have access to secrets on the project
		// The secrets api will return not authenticated with a message to authenticate that
		// we do not want to present to the user
		logging.Debug("Could not get secret definitions, got failure: %s", fail)
		return nil, locale.NewError("err_show_get_secrets", "Could not get secret definitions, you may not be authorized to view secrets on this project")
	}

	var userSecrets []string
	var projectSecrets []string
	for _, s := range sec {
		data := *s.Name
		if s.Description != "" {
			data = fmt.Sprintf("%s: %s", *s.Name, s.Description)
		}
		if strings.ToLower(*s.Scope) == "project" {
			projectSecrets = append(projectSecrets, data)
			continue
		}
		userSecrets = append(userSecrets, data)
	}

	if len(userSecrets) == 0 && len(projectSecrets) == 0 {
		return nil, nil
	}

	secrets := secretOutput{}
	if len(userSecrets) > 0 {
		secrets.User = userSecrets
	}
	if len(projectSecrets) > 0 {
		secrets.Project = projectSecrets
	}

	return &secrets, nil
}
