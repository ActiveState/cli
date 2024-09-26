package show

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/pkg/runtime_helpers"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/secrets"
	"github.com/ActiveState/cli/pkg/localcommit"
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

type primeable interface {
	primer.Projecter
	primer.Outputer
	primer.Conditioner
	primer.Auther
}

type RuntimeDetails struct {
	Name         string `json:"name" locale:"state_show_details_name,Name"`
	Organization string `json:"organization" locale:"state_show_details_organization,Organization"`
	NameSpace    string `json:"namespace" locale:"state_show_details_namespace,Namespace"`
	Location     string `json:"location" locale:"state_show_details_location,Location"`
	Executables  string `json:"executables" locale:"state_show_details_executables,Executables"`
	Visibility   string `json:"visibility" locale:"state_show_details_visibility,Visibility"`
	LastCommit   string `json:"last_commit" locale:"state_show_details_latest_commit,Latest Commit"`
}

type showOutput struct {
	output output.Outputer
	data   outputData
}

type outputData struct {
	ProjectURL string `json:"project_url" locale:"project_url,Project URL"`
	RuntimeDetails
	Platforms []platformRow     `json:"platforms"`
	Languages []languageRow     `json:"languages"`
	Secrets   *secretOutput     `json:"secrets" locale:"secrets,Secrets"`
	Events    []string          `json:"events,omitempty"`
	Scripts   map[string]string `json:"scripts,omitempty"`
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

func (o *showOutput) MarshalOutput(format output.Format) interface{} {
	o.output.Print(locale.Tl("show_details_intro", "Here are the details of your runtime environment.\n"))
	o.output.Print(
		struct {
			*RuntimeDetails `opts:"verticalTable"`
		}{&o.data.RuntimeDetails},
	)
	o.output.Print(output.Title(locale.Tl("state_show_events_header", "Events")))
	o.output.Print(formatSlice(o.data.Events))
	o.output.Print(output.Title(locale.Tl("state_show_scripts_header", "Scripts")))
	o.output.Print(formatScripts(o.data.Scripts))
	o.output.Print(output.Title(locale.Tl("state_show_platforms_header", "Platforms")))
	o.output.Print(o.data.Platforms)
	o.output.Print(output.Title(locale.Tl("state_show_languages_header", "Languages")))
	o.output.Print(o.data.Languages)

	return output.Suppress
}

func (o *showOutput) MarshalStructured(format output.Format) interface{} {
	return o.data
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

	var projectDir string
	if params.Remote != "" {
		namespaced, err := project.ParseNamespace(params.Remote)
		if err != nil {
			return locale.WrapError(err, "err_show_parse_namespace", "Invalid remote argument. It must be of the form <org/project>")
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
			return rationalize.ErrNoProject
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

		commitID, err = localcommit.Get(s.project.Dir())
		if err != nil {
			return errs.Wrap(err, "Unable to get local commit")
		}

		projectDir = filepath.Dir(s.project.Path())
		if fileutils.IsSymlink(projectDir) {
			projectDir, err = fileutils.ResolveUniquePath(projectDir)
			if err != nil {
				return locale.WrapError(err, "err_show_projectdir", "Could not resolve project directory symlink")
			}
		}
	}

	remoteProject, err := model.LegacyFetchProjectByName(owner, projectName)
	var errProjectNotFound *model.ErrProjectNotFound
	if err != nil && errors.As(err, &errProjectNotFound) {
		return locale.WrapError(err, "err_show_project_not_found", "Please run '[ACTIONABLE]state push[/RESET]' to synchronize this project with the ActiveState Platform.")
	} else if err != nil {
		return locale.WrapError(err, "err_show_get_project", "Could not get remote project details")
	}

	if projectURL == "" {
		projectURL = model.ProjectURL(owner, projectName, commitID.String())
	}

	platforms, err := platformsData(owner, projectName, commitID, s.auth)
	if err != nil {
		return locale.WrapError(err, "err_show_platforms", "Could not retrieve platform information")
	}

	languages, err := languagesData(commitID, s.auth)
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

	if projectDir != "" {
		rd.Location = projectDir
	}

	if params.Remote == "" {
		rd.Executables = runtime_helpers.ExecutorPathFromProject(s.project)
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

	s.out.Print(&showOutput{s.out, outputData})

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

func platformsData(owner, project string, branchID strfmt.UUID, auth *authentication.Auth) ([]platformRow, error) {
	remotePlatforms, err := model.FetchPlatformsForCommit(branchID, auth)
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

func languagesData(commitID strfmt.UUID, auth *authentication.Auth) ([]languageRow, error) {
	platformLanguages, err := model.FetchLanguagesForCommit(commitID, auth)
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

func commitsData(owner, project, branchName string, commitID strfmt.UUID, localProject *project.Project, auth *authentication.Auth) (string, error) {
	latestCommit, err := model.BranchCommitID(owner, project, branchName)
	if err != nil {
		return "", locale.WrapError(err, "err_show_get_latest_commit", "Could not get latest commit ID")
	}

	if !auth.Authenticated() {
		return latestCommit.String(), nil
	}

	belongs, err := model.CommitBelongsToBranch(owner, project, branchName, commitID, auth)
	if err != nil {
		return "", locale.WrapError(err, "err_show_get_commit_belongs", "Could not determine if commit belongs to branch")
	}

	if localProject != nil && localProject.Owner() == owner && localProject.Name() == project && belongs {
		var latestCommitID strfmt.UUID
		if latestCommit != nil {
			latestCommitID = *latestCommit
		}
		behind, err := model.CommitsBehind(latestCommitID, commitID, auth)
		if err != nil {
			return "", locale.WrapError(err, "err_show_commits_behind", "Could not determine number of commits behind latest")
		}
		localCommitID, err := localcommit.Get(localProject.Dir())
		if err != nil {
			return "", errs.Wrap(err, "Unable to get local commit")
		}
		if behind > 0 {
			return fmt.Sprintf("%s (%d %s)", localCommitID.String(), behind, locale.Tl("show_commits_behind_latest", "behind latest")), nil
		} else if behind < 0 {
			return fmt.Sprintf("%s (%d %s)", localCommitID.String(), -behind, locale.Tl("show_commits_ahead_of_latest", "ahead of latest")), nil
		}
		return localCommitID.String(), nil
	}

	return latestCommit.String(), nil
}

func secretsData(owner, project string, auth *authentication.Auth) (*secretOutput, error) {
	if !auth.Authenticated() {
		return nil, nil
	}

	client := secretsapi.Get(auth)
	sec, err := secrets.DefsByProject(client, owner, project)
	if err != nil {
		logging.Debug("Could not get secret definitions, got failure: %s", err)
		return nil, locale.WrapError(err, "err_show_get_secrets", "Could not get secret definitions. You may not be authorized to view secrets on this project")
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
