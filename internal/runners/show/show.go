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
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
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
}

type primeable interface {
	primer.Projecter
	primer.Outputer
	primer.Conditioner
}

type outputData struct {
	ProjectURL   string `locale:"project_url,Project URL"`
	Namespace    string
	Name         string
	Organization string
	Visibility   string `locale:"visibility,Visibility"`
	Commit       string `locale:"commit,Commit"`
	Platforms    []string
	Languages    []string
	Secrets      map[string][]string `locale:"secrets,Secrets"`
	Events       []string            `json:",omitempty"`
	Scripts      map[string]string   `json:",omitempty"`
}

// New returns a pointer to an instance of Show.
func New(prime primeable) *Show {
	return &Show{
		project:     prime.Project(),
		out:         prime.Output(),
		conditional: prime.Conditional(),
	}
}

// Run is the primary show logic.
func (s *Show) Run(params Params) error {
	logging.Debug("Execute show")

	var (
		owner       string
		projectName string
		events      []string
		scripts     map[string]string
		err         error
	)

	if params.Remote != "" {
		namespaced, fail := project.ParseNamespace(params.Remote)
		if fail != nil {
			return locale.WrapError(fail.ToError(), "err_show_parse_namespace", "Invalid remote argument, must be of the form <Owner>/<Project>")
		}

		owner = namespaced.Owner
		projectName = namespaced.Project
	} else {
		if s.project == nil {
			return locale.NewError("err_no_projectfile")
		}

		owner = s.project.Source().Owner
		projectName = s.project.Source().Name

		events, err = eventsData(s.project.Source(), s.conditional)
		if err != nil {
			return locale.WrapError(err, "err_show_events", "Could not parse events")
		}

		scripts, err = scriptsData(s.project.Source(), s.conditional)
		if err != nil {
			return locale.WrapError(err, "err_show_scripts", "Could not parse scripts")
		}
	}

	branch, fail := model.DefaultBranchForProjectName(owner, projectName)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_show_get_default_branch", "Could not get project information from the platform")
	}
	if branch.CommitID == nil {
		return locale.NewError("err_show_commitID", "Remote project details are incorrect. Default branch is missing commitID")
	}

	projectURL := model.ProjectURL(owner, projectName, branch.CommitID.String())

	platforms, err := platformsData(owner, projectName, *branch.CommitID)
	if err != nil {
		return locale.WrapError(err, "err_show_platforms", "Could not retrieve platform information")
	}

	languages, err := languagesData(owner, projectName)
	if err != nil {
		return locale.WrapError(err, "err_show_langauges", "Could not retrieve language information")
	}

	visibility, err := visibilityData(owner, projectName)
	if err != nil {
		return locale.WrapError(err, "err_show_visibility", "Could not get visibility information")
	}

	commit, err := commitsData(owner, projectName, *branch.CommitID, s.project)
	if err != nil {
		return locale.WrapError(err, "err_show_commit", "Could not get commit information")
	}

	secrets, err := secretsData(owner, projectName)
	if err != nil {
		return locale.WrapError(err, "err_show_secrets", "Could not get secret information")
	}

	data := outputData{
		ProjectURL:   projectURL,
		Namespace:    fmt.Sprintf("%s/%s", owner, projectName),
		Name:         projectName,
		Organization: owner,
		Visibility:   visibility,
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
		return nil, locale.WrapError(fail.ToError(), "err_show_get_platforms", "Could not get platform details for commit: {{.V0}}", branchID.String())
	}

	platforms := make([]string, len(remotePlatforms))
	for i, plat := range remotePlatforms {
		if plat.Kernel == nil || plat.Kernel.Name == nil {
			continue
		}
		platforms[i] = *plat.Kernel.Name
	}

	return platforms, nil
}

func languagesData(owner, project string) ([]string, error) {
	platformLanguages, fail := model.FetchLanguagesForProject(owner, project)
	if fail != nil {
		return nil, locale.WrapError(fail.ToError(), "err_show_get_langauges", "Could not get languages for project")
	}

	languages := make([]string, len(platformLanguages))
	for i, pl := range platformLanguages {
		languages[i] = fmt.Sprintf("%s-%s", pl.Name, pl.Version)
	}

	return languages, nil
}

func visibilityData(owner, project string) (string, error) {
	platfomProject, fail := model.FetchProjectByName(owner, project)
	if fail != nil {
		return "", locale.WrapError(fail.ToError(), "err_show_fetch_project", "Could not get remote project information")
	}

	if platfomProject.Private {
		return "Private", nil
	}
	return "Public", nil
}

func commitsData(owner, project string, commitID strfmt.UUID, localProject *project.Project) (string, error) {
	latestCommit, fail := model.LatestCommitID(owner, project)
	if fail != nil {
		return "", locale.WrapError(fail.ToError(), "err_show_get_latest_commit", "Could not get latest commit ID")
	}

	if localProject != nil && localProject.Owner() == owner && localProject.Name() == project {
		behind, fail := model.CommitsBehindLatest(owner, project, localProject.CommitID())
		if fail != nil {
			return "", locale.WrapError(fail.ToError(), "err_show_commits_behind", "Could not determine number of commits behind latest")
		}
		if behind != 0 {
			return fmt.Sprintf("%s (%d behind latest)", latestCommit, behind), nil
		}
	}

	return latestCommit.String(), nil
}

func secretsData(owner, project string) (map[string][]string, error) {
	client := secretsapi.Get()
	sec, fail := secrets.DefsByProject(client, owner, project)
	if fail != nil {
		return nil, locale.WrapError(fail.ToError(), "err_show_get_secrets", "Could not get secret definitions")
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

	secrets := make(map[string][]string)
	if len(userSecrets) > 0 {
		secrets["User"] = userSecrets
	}
	if len(projectSecrets) > 0 {
		secrets["Project"] = projectSecrets
	}

	if len(secrets) == 0 {
		return nil, nil
	}

	return secrets, nil
}
