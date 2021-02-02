package show

import (
	"fmt"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/secrets"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
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

type RuntimeDetails struct {
	Name         string `locale:"state_show_details_name,Name"`
	Organization string `locale:"state_show_details_organization,Organization"`
	NameSpace    string `locale:"state_show_details_namespace,Namespace"`
	Visibility   string `locale:"state_show_details_visibility,Visibility"`
	LastCommit   string `locale:"state_show_details_latest_commit,Latest Commit"`
}

type outputDataPrinter struct {
	output output.Outputer
	data   outputData
}
type outputData struct {
	ProjectURL string `locale:"project_url,Project URL"`
	RuntimeDetails
	Platforms []platformRow
	Languages []languageRow
	Secrets   *secretOutput     `locale:"secrets,Secrets"`
	Events    []string          `json:",omitempty"`
	Scripts   map[string]string `json:",omitempty"`
}

func formatScripts(scripts map[string]string) string {
	var res []string

	for k, v := range scripts {
		res = append(res, fmt.Sprintf("• %s", k))
		if v != "" {
			res = append(res, fmt.Sprintf("  └─  %s", v))
		}
	}
	return strings.Join(res, "\n")
}

func formatSlice(slice []string) string {
	var res []string

	for _, v := range slice {
		res = append(res, fmt.Sprintf("• %s", v))
	}
	return strings.Join(res, "\n")
}

func (od *outputDataPrinter) MarshalOutput(format output.Format) interface{} {
	if format != output.PlainFormatName {
		return od.data
	}

	od.output.Print(locale.Tl("show_details_intro", "Here are the details of your runtime environment.\n"))
	od.output.Print(
		struct {
			*RuntimeDetails `opts:"verticalTable"`
		}{&od.data.RuntimeDetails},
	)
	od.output.Print(output.Heading(locale.Tl("state_show_events_header", "Events")))
	od.output.Print(formatSlice(od.data.Events))
	od.output.Print(output.Heading(locale.Tl("state_show_scripts_header", "Scripts")))
	od.output.Print(formatScripts(od.data.Scripts))
	od.output.Print(output.Heading(locale.Tl("state_show_platforms_header", "Platforms")))
	od.output.Print(od.data.Platforms)
	od.output.Print(output.Heading(locale.Tl("state_show_languages_header", "Languages")))
	od.output.Print(od.data.Languages)

	return output.Suppress
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
		commitID    strfmt.UUID
		branchName  string
		events      []string
		scripts     map[string]string
		err         error
	)

	if params.Remote != "" {
		namespaced, err := project.ParseNamespace(params.Remote)
		if err != nil {
			return locale.WrapError(err, "err_show_parse_namespace", "Invalid remote argument, must be of the form <Owner>/<Project>")
		}

		owner = namespaced.Owner
		projectName = namespaced.Project

		branch, err := model.DefaultBranchForProjectName(owner, projectName)
		if err != nil {
			return locale.WrapError(err, "err_show_get_default_branch", "Could not get project information from the platform")
		}
		if branch.CommitID == nil {
			return locale.NewError("err_show_commitID", "Remote project details are incorrect. Default branch is missing commitID")
		}
		branchName = branch.Label
		commitID = *branch.CommitID
	} else {
		if s.project == nil {
			return locale.NewInputError("err_no_project")
		}

		if s.project.IsHeadless() {
			return locale.NewInputError("err_show_not_supported_headless", "This is not supported while in a headless state. Please visit {{.V0}} to create your project.", s.project.URL())
		}

		owner = s.project.Owner()
		projectName = s.project.Name()
		projectURL = s.project.URL()
		branchName = s.project.BranchName()

		events, err = eventsData(s.project.Source(), s.conditional)
		if err != nil {
			return locale.WrapError(err, "err_show_events", "Could not parse events")
		}

		scripts, err = scriptsData(s.project.Source(), s.conditional)
		if err != nil {
			return locale.WrapError(err, "err_show_scripts", "Could not parse scripts")
		}

		commitID = strfmt.UUID(s.project.CommitID())
	}

	remoteProject, err := model.FetchProjectByName(owner, projectName)
	if err != nil && errs.Matches(err, &model.ErrProjectNotFound{}) {
		return locale.WrapError(err, "err_show_project_not_found", "Please run `state push` to synchronize this project with the ActiveState Platform.")
	} else if err != nil {
		return locale.WrapError(err, "err_show_get_project", "Could not get remote project details")
	}

	if projectURL == "" {
		projectURL = model.ProjectURL(owner, projectName, commitID.String())
	}

	platforms, err := platformsData(owner, projectName, commitID)
	if err != nil {
		return locale.WrapError(err, "err_show_platforms", "Could not retrieve platform information")
	}

	languages, err := languagesData(commitID)
	if err != nil {
		return locale.WrapError(err, "err_show_langauges", "Could not retrieve language information")
	}

	commit, err := commitsData(owner, projectName, branchName, commitID, s.project, s.auth)
	if err != nil {
		return locale.WrapError(err, "err_show_commit", "Could not get commit information")
	}

	secrets, err := secretsData(owner, projectName, s.auth)
	if err != nil {
		return locale.WrapError(err, "err_show_secrets", "Could not get secret information")
	}

	rd := RuntimeDetails{
		NameSpace:    fmt.Sprintf("%s/%s", owner, projectName),
		Name:         projectName,
		Organization: owner,
		Visibility:   visibilityData(owner, projectName, remoteProject),
		LastCommit:   commit,
	}

	outputData := outputData{
		ProjectURL:     projectURL,
		RuntimeDetails: rd,
		Languages:      languages,
		Platforms:      platforms,
		Secrets:        secrets,
		Events:         events,
		Scripts:        scripts,
	}

	odp := &outputDataPrinter{s.out, outputData}
	s.out.Print(odp)

	return nil
}

type platformRow struct {
	Name     string `json:"name" locale:"state_show_platform_name,Name"`
	Version  string `json:"version" locale:"state_show_platform_version,Version"`
	BitWidth string `json:"bit_width" locale:"state_show_platform_bitwidth,Bit Width"`
}

type languageRow struct {
	Name    string `json:"name" locale:"state_show_language_name,Name"`
	Version string `json:"version" locale:"state_show_language_version,Version"`
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

func platformsData(owner, project string, branchID strfmt.UUID) ([]platformRow, error) {
	remotePlatforms, err := model.FetchPlatformsForCommit(branchID)
	if err != nil {
		return nil, locale.WrapError(err, "err_show_get_platforms", "Could not get platform details for commit: {{.V0}}", branchID.String())
	}

	platforms := make([]platformRow, 0, len(remotePlatforms))
	for _, plat := range remotePlatforms {
		if plat.DisplayName != nil {
			p := platformRow{Name: *plat.OperatingSystem.Name, Version: *plat.OperatingSystemVersion.Version, BitWidth: *plat.CPUArchitecture.BitWidth}
			platforms = append(platforms, p)
		}
	}

	return platforms, nil
}

func languagesData(commitID strfmt.UUID) ([]languageRow, error) {
	platformLanguages, err := model.FetchLanguagesForCommit(commitID)
	if err != nil {
		return nil, locale.WrapError(err, "err_show_get_languages", "Could not get languages for project")
	}

	languages := make([]languageRow, 0, len(platformLanguages))
	for _, pl := range platformLanguages {
		l := languageRow{Name: pl.Name, Version: pl.Version}
		languages = append(languages, l)
	}

	return languages, nil
}

func visibilityData(owner, project string, remoteProject *mono_models.Project) string {
	if remoteProject.Private {
		return locale.T("private")
	}
	return locale.T("public")
}

func commitsData(owner, project, branchName string, commitID strfmt.UUID, localProject *project.Project, auth auther) (string, error) {
	latestCommit, err := model.BranchCommitID(owner, project, branchName)
	if err != nil {
		return "", locale.WrapError(err, "err_show_get_latest_commit", "Could not get latest commit ID")
	}

	if !auth.Authenticated() {
		return latestCommit.String(), nil
	}

	if localProject != nil && localProject.Owner() == owner && localProject.Name() == project {
		var latestCommitID strfmt.UUID
		if latestCommit != nil {
			latestCommitID = *latestCommit
		}
		behind, err := model.CommitsBehind(latestCommitID, commitID)
		if err != nil {
			return "", locale.WrapError(err, "err_show_commits_behind", "Could not determine number of commits behind latest")
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
	sec, err := secrets.DefsByProject(client, owner, project)
	if err != nil {
		logging.Debug("Could not get secret definitions, got failure: %s", err)
		return nil, locale.WrapError(err, "err_show_get_secrets", "Could not get secret definitions, you may not be authorized to view secrets on this project")
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
