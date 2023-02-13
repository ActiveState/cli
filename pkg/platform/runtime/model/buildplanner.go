package model

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
	vcsModel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/go-openapi/strfmt"
	"github.com/machinebox/graphql"
)

// HostPlatform stores a reference to current platform
var HostPlatform string

func init() {
	HostPlatform = sysinfo.OS().String()
	if osName, ok := os.LookupEnv(constants.OverrideOSNameEnvVarName); ok {
		HostPlatform = osName
	}
}

type BuildPlannerError struct {
	wrapped          error
	validationErrors []string
	isTransient      bool
}

func (e *BuildPlannerError) Error() string {
	return "resolve_err"
}

func (e *BuildPlannerError) Unwrap() error {
	return e.wrapped
}

func (e *BuildPlannerError) ValidationErrors() []string {
	return e.validationErrors
}

func (e *BuildPlannerError) IsTransient() bool {
	return e.isTransient
}

type BuildPlanner struct {
	auth   *authentication.Auth
	client *gqlclient.Client
	def    *Recipe
}

func NewBuildPlanner(auth *authentication.Auth) *BuildPlanner {
	bpURL := constants.APIBuildPlannerURL
	if url, ok := os.LookupEnv("_TEST_BUILDPLAN_URL"); ok {
		bpURL = url
	}
	logging.Debug("Using build planner at: %s", bpURL)

	client := gqlclient.NewWithOpts(bpURL, 0, graphql.WithHTTPClient(&http.Client{}))

	if auth.Authenticated() {
		client.SetTokenProvider(auth)
	}

	return &BuildPlanner{
		auth:   auth,
		client: client,
		def:    NewRecipe(auth),
	}
}

func (bp *BuildPlanner) FetchBuildResult(commitID strfmt.UUID, owner, project string) (*BuildResult, error) {
	resp := &model.BuildPlan{}
	err := bp.client.Run(request.BuildPlan(owner, project, commitID.String()), resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch build plan")
	}

	// This is a lot of awkward error checking
	// This error checking should go away with the new commit query
	// TODO: Investigate commit not found errors
	if resp.Project.Type == model.ProjectResultNotFound {
		return nil, locale.NewError("err_buildplanner_project_not_found", "Build plan does not contain project")
	}
	if resp.Project.Commit.Type == model.CommitResultNotFound {
		return nil, locale.NewError("err_buildplanner_commit_not_found", "Build plan does not contain commit")
	}
	if resp.Project.Commit.Build.Type == model.BuildResultPlanningError {
		var errs []string
		var isTransient bool
		for _, se := range resp.Project.Commit.Build.SubErrors {
			errs = append(errs, se.Message)
			isTransient = se.IsTransient
		}
		return nil, &BuildPlannerError{
			wrapped:          locale.NewError("err_buildplanner", resp.Project.Commit.Build.Error),
			validationErrors: errs,
			isTransient:      isTransient,
		}
	}

	// The type aliasing in the query populates the
	// response with emtpy targets that we have to remove
	removeEmptyTargets(resp)

	var bpPlatforms []strfmt.UUID
	for _, t := range resp.Project.Commit.Build.Terminals {
		if t.Tag == model.TagOrphan {
			continue
		}
		bpPlatforms = append(bpPlatforms, strfmt.UUID(strings.TrimPrefix(t.Tag, "platform:")))
	}

	platformID, err := platformModel.FilterCurrentPlatform(HostPlatform, bpPlatforms)
	if err != nil {
		return nil, locale.WrapError(err, "err_filter_current_platform")
	}

	var filteredTerminals []*model.NamedTarget
	for _, t := range resp.Project.Commit.Build.Terminals {
		if platformID.String() == strings.TrimPrefix(t.Tag, "platform:") {
			filteredTerminals = append(filteredTerminals, t)
		}
	}
	resp.Project.Commit.Build.Terminals = filteredTerminals

	buildEngine := Alternative
	for _, s := range resp.Project.Commit.Build.Sources {
		if s.Namespace == "builder" && s.Name == "camel" {
			buildEngine = Camel
		}
	}

	res := BuildResult{
		BuildEngine: buildEngine,
		Build:       resp.Project.Commit.Build,
		BuildReady:  resp.Project.Commit.Build.Status == model.Ready,
	}

	if resp.Project.Commit.Build.Status == model.Building {
		// If the build is alternative the ID type will identify it as a recipe ID.
		// The other buildLogID type is for camel builds which we don't use for
		// builds in progress.
		for _, id := range resp.Project.Commit.Build.BuildLogIDs {
			if id.Type == model.BuildLogRecipeID {
				res.RecipeID = strfmt.UUID(id.ID)
			}
		}
	}

	return &res, nil
}

func removeEmptyTargets(bp *model.BuildPlan) {
	var steps []*model.Step
	for _, step := range bp.Project.Commit.Build.Steps {
		if step.TargetID == "" {
			continue
		}
		steps = append(steps, step)
	}

	var sources []*model.Source
	for _, source := range bp.Project.Commit.Build.Sources {
		if source.TargetID == "" {
			continue
		}
		sources = append(sources, source)
	}

	var artifacts []*model.Artifact
	for _, artifact := range bp.Project.Commit.Build.Artifacts {
		if artifact.TargetID == "" {
			continue
		}
		artifacts = append(artifacts, artifact)
	}

	bp.Project.Commit.Build.Steps = steps
	bp.Project.Commit.Build.Sources = sources
	bp.Project.Commit.Build.Artifacts = artifacts
}

type PushCommitParams struct {
	Owner            string
	Project          string
	ParentCommit     string
	Description      string
	BranchRef        string
	PackageName      string
	PackageVersion   string
	PackageNamespace vcsModel.Namespace
	Operation        model.Operation
	Time             time.Time
}

func (bp *BuildPlanner) PushCommit(params *PushCommitParams) (string, error) {
	// If parent commit is provided then get the build graph
	// If it is not create a blank build graph
	var err error
	script := model.NewBuildScript()
	if params.ParentCommit != "" {
		script, err = bp.GetBuildScript(params.Owner, params.Project, params.ParentCommit)
		if err != nil {
			return "", errs.Wrap(err, "Failed to get build graph")
		}
	}

	requirement := model.Requirement{
		Namespace: params.PackageNamespace.String(),
		Name:      params.PackageName,
	}

	if params.PackageVersion != "" {
		requirement.VersionRequirement = []model.VersionRequirement{{model.ComparatorEQ: params.PackageVersion}}
	}

	// Call the build graph update function with the operation
	script, err = script.Update(params.Operation, []model.Requirement{requirement})
	if err != nil {
		return "", errs.Wrap(err, "Failed to update build graph")
	}
	script.Let.Runtime.SolveLegacy.AtTime = params.Time.Format(time.RFC3339)

	// With the updated build graph call the save and build mutation
	request := request.PushCommit(params.Owner, params.Project, params.ParentCommit, params.BranchRef, params.Description, script)
	resp := &model.PushCommitResult{}
	err = bp.client.Run(request, resp)
	if err != nil {
		return "", errs.Wrap(err, "failed to fetch build plan")
	}

	if resp.NotFound != nil {
		return "", errs.New("Commit not found: %s", resp.NotFound.Message)
	}

	if resp.Error != nil {
		return "", errs.New("PushCommit failed: %s", resp.Error.Message)
	}

	return resp.Commit.CommitID, nil
}

func (bp *BuildPlanner) GetBuildScript(owner, project, commitID string) (*model.BuildScript, error) {
	resp := &model.BuildPlan{}
	err := bp.client.Run(request.BuildScript(owner, project, commitID), resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch build graph")
	}

	if resp.Project == nil {
		return nil, errs.New("Project not found")
	}

	if resp.Project.Commit == nil {
		return nil, errs.New("Commit not found")
	}

	if resp.Project.Commit.Script == nil {
		return nil, errs.New("Commit script not found")
	}

	return resp.Project.Commit.Script, nil
}
