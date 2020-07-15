package show

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/secrets"
	"github.com/ActiveState/cli/internal/updater"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	prj "github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
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
	// TODO: This does not tab properly in output
	Secrets *secretsOutputData `locale:"secrets,Secrets"`
	Events  []string           `json:",omitempty"`
	Scripts map[string]string  `json:",omitempty"`
}

type secretsOutputData struct {
	User    []string `locale:"user,User"`
	Project []string `locale:"project,Project"`
}

// New returns a pointer to an instance of Show.
func New(prime primeable) *Show {
	return &Show{
		prime.Project(),
		prime.Output(),
		prime.Conditional(),
	}
}

// Run is the primary show logic.
func (s *Show) Run(params Params) error {
	logging.Debug("Execute")

	if s.project == nil {
		return locale.NewError("err_no_projectfile")
	}

	pj := s.project
	namespace := fmt.Sprintf("%s/%s", s.project.Owner(), s.project.Name())
	if params.Remote != "" {
		path := params.Remote
		projectFilePath := filepath.Join(params.Remote, constants.ConfigFileName)

		if _, err := os.Stat(path); err != nil {
			return locale.WrapError(
				err,
				"err_state_show_path_does_not_exist",
				"Directory does not exist.",
			)
		}

		if _, err := os.Stat(projectFilePath); err != nil {
			return locale.WrapError(
				err,
				"err_state_show_no_config",
				"activestate.yaml file not found in the given location.",
			)
		}

		projectFile, fail := projectfile.Parse(projectFilePath)
		if fail != nil {
			logging.Errorf("Unable to parse activestate.yaml: %s", fail)
			return locale.WrapError(
				fail,
				"err_state_show_project_parse",
				"Could not parse activestate.yaml.",
			)
		}

		pj, fail = prj.New(projectFile)
		if fail != nil {
			return fail.ToError()
		}

		split := strings.Split(filepath.Clean(params.Remote), string(filepath.Separator))
		namespace = fmt.Sprintf("%s/%s", split[len(split)-2], split[len(split)-1])
	}

	src := pj.Source()

	updater.PrintUpdateMessage(src.Path())

	namespaced, fail := prj.ParseNamespace(namespace)
	if fail != nil {
		return locale.WrapError(
			fail.ToError(),
			"err_show_parse_namespace",
			"Could not parse remote project namespace",
		)
	}

	events, err := eventsData(src, s.conditional)
	if err != nil {
		return locale.WrapError(err, "err_show_events", "Could not parse events.")
	}

	scripts, err := scriptsData(src, s.conditional)
	if err != nil {
		return locale.WrapError(err, "err_show_scripts", "Could not parse scripts.")
	}

	platforms, err := platformsData(namespaced.Owner, namespaced.Project)
	if err != nil {
		return locale.WrapError(err, "err_show_platforms", "Could not retrieve platform information")
	}

	languages, err := languagesData(namespaced.Owner, namespaced.Project)
	if err != nil {
		return locale.WrapError(err, "err_show_langauges", "Could not retrieve language information")
	}

	visibility, err := visibilityData(namespaced.Owner, namespaced.Project)
	if err != nil {
		return locale.WrapError(err, "err_show_visibility", "Could not get visibility information")
	}

	commit, err := s.commitsData(namespaced.Owner, namespaced.Project)
	if err != nil {
		return locale.WrapError(err, "err_show_commit", "Could not get commit information")
	}

	secrets, err := secretsData(namespaced.Owner, namespaced.Project)
	if err != nil {
		return locale.WrapError(err, "err_show_secrets", "Could not get secret information")
	}

	data := outputData{
		ProjectURL:   pj.Source().Project,
		Namespace:    pj.Namespace(),
		Name:         pj.Name(),
		Organization: pj.Owner(),
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

func platformsData(owner, project string) ([]string, error) {
	branch, fail := model.DefaultBranchForProjectName(owner, project)
	if fail != nil {
		return nil, locale.WrapError(fail.ToError(), "err_show_get_default_branch", "Could not get the default branch")
	}
	if branch.CommitID == nil {
		return nil, locale.NewError("err_show_get_commitID", "Could not get commit ID for default branch")
	}

	// TODO: Could create model method for getting platforms via project namespace
	remotePlatforms, fail := model.FetchPlatformsForCommit(*branch.CommitID)
	if fail != nil {
		return nil, locale.WrapError(fail.ToError(), "err_show_get_platforms", "Could not get platform details")
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
	// TODO: Improve all error messages
	platformLanguages, fail := model.FetchLanguagesForProject(owner, project)
	if fail != nil {
		return nil, locale.WrapError(fail.ToError(), "err_show_get_langauges", "Could not get languages for proejct: {{.V0}}", project)
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
		return "", locale.WrapError(fail.ToError(), "err_show_fetch_project", "Could not get project: {{.V0}}/{{.V1}}", owner, project)
	}

	if platfomProject.Private {
		return "Private", nil
	}
	return "Public", nil
}

func (s *Show) commitsData(owner, project string) (string, error) {
	// TODO: Store this, or similar information in the show struct
	branch, fail := model.DefaultBranchForProjectName(owner, project)
	if fail != nil {
		return "", locale.WrapError(fail.ToError(), "err_show_get_default_branch", "Could not get the default branch")
	}
	if branch.CommitID == nil {
		return "", locale.NewError("err_show_get_commitID", "Could not get commit ID for default branch")
	}

	latestCommit, fail := model.LatestCommitID(owner, project)
	if fail != nil {
		return "", locale.WrapError(fail.ToError(), "err_show_get_latest_commit", "Could not get latest commit ID")
	}

	behind, fail := model.CommitsBehindLatest(owner, project, s.project.CommitID())
	if fail != nil {
		return "", locale.WrapError(fail.ToError(), "err_show_commits_behind", "Could not get commits behind latest")
	}
	if behind != 0 {
		return fmt.Sprintf("%s (%d behind latest)", latestCommit, behind), nil
	}

	return fmt.Sprintf("%s", latestCommit), nil
}

func secretsData(owner, project string) (*secretsOutputData, error) {
	client := secretsapi.Get()
	sec, fail := secrets.DefsByProject(client, owner, project)
	if fail != nil {
		return nil, locale.WrapError(fail.ToError(), "err_show_get_secrets", "Could not get secrets for project: {{.V0}}/{{.V1}}", owner, project)
	}

	var userSecrets []string
	var projectSecrets []string
	for _, s := range sec {
		data := *s.Name
		if s.Description != "" {
			data = fmt.Sprintf("%s: %s", *s.Name, s.Description)
		}
		if *s.Scope == "project" {
			projectSecrets = append(projectSecrets, data)
			continue
		}
		userSecrets = append(userSecrets, data)
	}

	return &secretsOutputData{
		User:    userSecrets,
		Project: projectSecrets,
	}, nil
}
