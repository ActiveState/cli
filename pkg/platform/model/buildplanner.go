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
	"github.com/ActiveState/cli/pkg/platform/api"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/go-openapi/strfmt"
	"github.com/machinebox/graphql"
)

const (
	pollInterval = 1 * time.Second
	pollTimeout  = 30 * time.Second
)

// HostPlatform stores a reference to current platform
var HostPlatform string

func init() {
	HostPlatform = sysinfo.OS().String()
	if osName, ok := os.LookupEnv(constants.OverrideOSNameEnvVarName); ok {
		HostPlatform = osName
	}
}

// BuildResult is the unified response of a Build request
type BuildResult struct {
	BuildEngine         BuildEngine
	RecipeID            strfmt.UUID
	CommitID            strfmt.UUID
	Build               *bpModel.Build
	BuildStatusResponse *headchef_models.V1BuildStatusResponse
	BuildStatus         headchef.BuildStatusEnum
	BuildReady          bool
}

func (b *BuildResult) OrderedArtifacts() []artifact.ArtifactID {
	res := make([]artifact.ArtifactID, 0, len(b.Build.Artifacts))
	for _, a := range b.Build.Artifacts {
		res = append(res, a.TargetID)
	}
	return res
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
}

func NewBuildPlanModel(auth *authentication.Auth) *BuildPlanner {
	bpURL := api.GetServiceURL(api.ServiceBuildPlanner).String()
	logging.Debug("Using build planner at: %s", bpURL)

	client := gqlclient.NewWithOpts(bpURL, 0, graphql.WithHTTPClient(&http.Client{}))

	if auth.Authenticated() {
		client.SetTokenProvider(auth)
	}

	return &BuildPlanner{
		auth:   auth,
		client: client,
	}
}

func (bp *BuildPlanner) FetchBuildResult(commitID strfmt.UUID, owner, project string) (*BuildResult, error) {
	resp := &bpModel.BuildPlan{}
	err := bp.client.Run(request.BuildPlan(owner, project, commitID.String()), resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch build plan")
	}

	// Check for errors in the response
	if resp.Project.Type == bpModel.NotFound {
		return nil, locale.NewError("err_buildplanner_project_not_found", "Build plan does not contain project")
	}
	if resp.Project.Commit.Type == bpModel.NotFound {
		return nil, locale.NewError("err_buildplanner_commit_not_found", "Build plan does not contain commit")
	}
	if resp.Project.Commit.Build.Type == bpModel.BuildResultPlanningError {
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

	// The BuildPlanner will return a build plan with a status of
	// "planning" if the build plan is not ready yet. We need to
	// poll the BuildPlanner until the build is ready.
	if resp.Project.Commit.Build.Status == bpModel.Planning {
		resp, err = bp.pollBuildPlan(owner, project, commitID.String())
		if err != nil {
			return nil, errs.Wrap(err, "failed to poll build plan")
		}
	}

	// The type aliasing in the query populates the
	// response with emtpy targets that we should remove
	removeEmptyTargets(resp)

	// Extract the available platforms from the build plan
	var bpPlatforms []strfmt.UUID
	for _, t := range resp.Project.Commit.Build.Terminals {
		if t.Tag == bpModel.TagOrphan {
			continue
		}
		bpPlatforms = append(bpPlatforms, strfmt.UUID(strings.TrimPrefix(t.Tag, "platform:")))
	}

	// Get the platform ID for the current platform
	platformID, err := FilterCurrentPlatform(HostPlatform, bpPlatforms)
	if err != nil {
		return nil, locale.WrapError(err, "err_filter_current_platform")
	}

	// Filter the build terminals to only include the current platform
	var filteredTerminals []*bpModel.NamedTarget
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
			break
		}
	}

	res := BuildResult{
		BuildEngine: buildEngine,
		Build:       resp.Project.Commit.Build,
		BuildReady:  resp.Project.Commit.Build.Status == bpModel.Ready,
		CommitID:    strfmt.UUID(resp.Project.Commit.CommitID),
	}

	// We want to extract the recipe ID from the BuildLogIDs.
	// We do this because if the build is in progress we will need to reciepe ID to
	// initialize the build log streamer.
	// For camel builds the ID type will not be BuildLogRecipeID but this is okay
	// because the state tool does not display in progress information for camel builds.
	for _, id := range resp.Project.Commit.Build.BuildLogIDs {
		if id.Type == bpModel.BuildLogRecipeID {
			if res.RecipeID != "" {
				return nil, errs.Wrap(err, "Build plan contains multiple recipe IDs")
			}
			res.RecipeID = strfmt.UUID(id.ID)
		}
	}

	return &res, nil
}

func (bp *BuildPlanner) pollBuildPlan(owner, project, commitID string) (*bpModel.BuildPlan, error) {
	var resp *bpModel.BuildPlan
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-ticker.C:
			err := bp.client.Run(request.BuildPlan(owner, project, commitID), resp)
			if err != nil {
				return nil, errs.Wrap(err, "failed to fetch build plan")
			}
			if resp.Project.Commit.Build.Status != bpModel.Planning {
				return resp, nil
			}
		case <-time.After(pollTimeout):
			return nil, locale.NewError("err_buildplanner_timeout", "Timed out waiting for build plan")
		}
	}
}

func removeEmptyTargets(bp *bpModel.BuildPlan) {
	var steps []*bpModel.Step
	for _, step := range bp.Project.Commit.Build.Steps {
		if step.TargetID == "" {
			continue
		}
		steps = append(steps, step)
	}

	var sources []*bpModel.Source
	for _, source := range bp.Project.Commit.Build.Sources {
		if source.TargetID == "" {
			continue
		}
		sources = append(sources, source)
	}

	var artifacts []*bpModel.Artifact
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

type StageCommitParams struct {
	Owner            string
	Project          string
	ParentCommit     string
	PackageName      string
	PackageVersion   string
	PackageNamespace Namespace
	Operation        bpModel.Operation
	TimeStamp        *strfmt.DateTime
}

func (bp *BuildPlanner) StageCommit(params StageCommitParams) (strfmt.UUID, error) {
	var err error
	script, err := bp.GetBuildExpression(params.Owner, params.Project, params.ParentCommit)
	if err != nil {
		return "", errs.Wrap(err, "Failed to get build graph")
	}

	requirement := bpModel.Requirement{
		Namespace: params.PackageNamespace.String(),
		Name:      params.PackageName,
	}

	if params.PackageVersion != "" {
		requirement.VersionRequirement = []bpModel.VersionRequirement{{bpModel.ComparatorEQ: params.PackageVersion}}
	}

	err = script.Update(params.Operation, requirement, *params.TimeStamp)
	if err != nil {
		return "", errs.Wrap(err, "Failed to update build graph")
	}

	// With the updated build expression call the stage commit mutation
	request := request.StageCommit(params.Owner, params.Project, params.ParentCommit, script)
	resp := &bpModel.StageCommitResult{}
	err = bp.client.Run(request, resp)
	if err != nil {
		return "", errs.Wrap(err, "failed to fetch build plan")
	}

	if resp.NotFoundError != nil {
		return "", errs.New("Commit not found: %s", resp.NotFoundError.Message)
	}

	if resp.Commit.Build.Status == bpModel.Planning {
		buildResult, err := bp.FetchBuildResult(strfmt.UUID(resp.Commit.CommitID), params.Owner, params.Project)
		if err != nil {
			return "", errs.Wrap(err, "failed to fetch build result")
		}

		return buildResult.CommitID, nil
	}

	return strfmt.UUID(resp.Commit.CommitID), nil
}

func (bp *BuildPlanner) GetBuildExpression(owner, project, commitID string) (*bpModel.BuildExpression, error) {
	resp := &bpModel.BuildPlan{}
	err := bp.client.Run(request.BuildExpression(owner, project, commitID), resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch build graph")
	}

	if resp.Project.Type == bpModel.NotFound {
		return nil, errs.New("Project not found: %s", resp.Project.Message)
	}
	if resp.Project.Commit.Type == bpModel.NotFound {
		return nil, errs.New("Commit not found: %s", resp.Project.Commit.Message)
	}

	expression, err := bpModel.NewBuildExpression(resp.Project.Commit.Script)
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse build expression")
	}

	return expression, nil
}
