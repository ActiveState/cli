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
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
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
		res = append(res, a.NodeID)
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

func NewBuildPlannerModel(auth *authentication.Auth) *BuildPlanner {
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
	logging.Debug("FetchBuildResult")
	resp := &bpModel.BuildPlan{}
	err := bp.client.Run(request.BuildPlan(commitID.String(), owner, project), resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch build plan")
	}

	if resp.Commit == nil {
		return nil, errs.New("Staged commit is nil")
	}

	// Check for errors in the response
	if resp.Commit.Type == bpModel.NotFound {
		return nil, locale.NewError("err_buildplanner_commit_not_found", "Build plan does not contain commit")
	}

	if resp.Commit.Build == nil {
		if resp.Commit.NotFoundError != nil {
			return nil, errs.New("Commit not found: %s", resp.Commit.NotFoundError.Message)
		}
		return nil, errs.New("Commit does not contain build")
	}

	if resp.Commit.Build.PlanningError != nil {
		var errs []string
		var isTransient bool
		for _, se := range resp.Commit.Build.SubErrors {
			errs = append(errs, se.Message)
			isTransient = se.IsTransient
		}
		return nil, &BuildPlannerError{
			wrapped:          locale.NewInputError("err_buildplanner", resp.Commit.Build.Message),
			validationErrors: errs,
			isTransient:      isTransient,
		}
	}

	// The BuildPlanner will return a build plan with a status of
	// "planning" if the build plan is not ready yet. We need to
	// poll the BuildPlanner until the build is ready.
	if resp.Commit.Build.Status == bpModel.Planning {
		resp, err = bp.pollBuildPlan(commitID.String(), owner, project)
		if err != nil {
			return nil, errs.Wrap(err, "failed to poll build plan")
		}
	}

	// The type aliasing in the query populates the
	// response with emtpy targets that we should remove
	removeEmptyTargets(resp)

	// Extract the available platforms from the build plan
	var bpPlatforms []strfmt.UUID
	for _, t := range resp.Commit.Build.Terminals {
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
	for _, t := range resp.Commit.Build.Terminals {
		if platformID.String() == strings.TrimPrefix(t.Tag, "platform:") {
			filteredTerminals = append(filteredTerminals, t)
		}
	}
	resp.Commit.Build.Terminals = filteredTerminals

	buildEngine := Alternative
	for _, s := range resp.Commit.Build.Sources {
		if s.Namespace == "builder" && s.Name == "camel" {
			buildEngine = Camel
			break
		}
	}

	res := BuildResult{
		BuildEngine: buildEngine,
		Build:       resp.Commit.Build,
		BuildReady:  resp.Commit.Build.Status == bpModel.Completed,
		CommitID:    strfmt.UUID(resp.Commit.CommitID),
	}

	// We want to extract the recipe ID from the BuildLogIDs.
	// We do this because if the build is in progress we will need to reciepe ID to
	// initialize the build log streamer.
	// This information will only be populated if the build is an alternate build.
	// This is specified in the build planner queries.
	for _, id := range resp.Commit.Build.BuildLogIDs {
		if res.RecipeID != "" {
			return nil, errs.Wrap(err, "Build plan contains multiple recipe IDs")
		}
		res.RecipeID = strfmt.UUID(id.ID)
	}

	return &res, nil
}

func (bp *BuildPlanner) pollBuildPlan(commitID, owner, project string) (*bpModel.BuildPlan, error) {
	var resp *bpModel.BuildPlan
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-ticker.C:
			err := bp.client.Run(request.BuildPlan(commitID, owner, project), resp)
			if err != nil {
				return nil, errs.Wrap(err, "failed to fetch build plan")
			}

			if resp == nil {
				continue
			}

			// This should not happen, but if it does we want to know and prevent
			// a potential panic below.
			if resp.Commit.Type == bpModel.NotFound {
				return nil, locale.NewError("err_buildplanner_commit_not_found", "Build plan does not contain commit")
			}

			if resp.Commit.Build.Status != bpModel.Planning {
				return resp, nil
			}
		case <-time.After(pollTimeout):
			return nil, locale.NewError("err_buildplanner_timeout", "Timed out waiting for build plan")
		}
	}
}

func removeEmptyTargets(bp *bpModel.BuildPlan) {
	var steps []*bpModel.Step
	for _, step := range bp.Commit.Build.Steps {
		if step.StepID == "" {
			continue
		}
		steps = append(steps, step)
	}

	var sources []*bpModel.Source
	for _, source := range bp.Commit.Build.Sources {
		if source.NodeID == "" {
			continue
		}
		sources = append(sources, source)
	}

	var artifacts []*bpModel.Artifact
	for _, artifact := range bp.Commit.Build.Artifacts {
		if artifact.NodeID == "" {
			continue
		}
		artifacts = append(artifacts, artifact)
	}

	bp.Commit.Build.Steps = steps
	bp.Commit.Build.Sources = sources
	bp.Commit.Build.Artifacts = artifacts
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
	logging.Debug("StageCommit")
	var err error
	expression, err := bp.GetBuildExpression(params.Owner, params.Project, params.ParentCommit)
	if err != nil {
		return "", errs.Wrap(err, "Failed to get build expression")
	}

	requirement := bpModel.Requirement{
		Namespace: params.PackageNamespace.String(),
		Name:      params.PackageName,
	}

	if params.PackageVersion != "" {
		requirement.VersionRequirement = []bpModel.VersionRequirement{{bpModel.VersionRequirementComparatorKey: bpModel.ComparatorEQ, bpModel.VersionRequirementVersionKey: params.PackageVersion}}
	}

	err = expression.Update(params.Operation, requirement, *params.TimeStamp)
	if err != nil {
		return "", errs.Wrap(err, "Failed to update build graph")
	}

	// With the updated build expression call the stage commit mutation
	request := request.StageCommit(params.Owner, params.Project, params.ParentCommit, expression)
	resp := &bpModel.StageCommitResult{}
	err = bp.client.Run(request, resp)
	if err != nil {
		return "", locale.WrapError(err, "err_buildplanner_stage_commit", "Failed to stage commit, error: {{.V0}}", err.Error())
	}

	if resp.Commit == nil {
		return "", errs.New("Staged commit is nil")
	}

	if resp.Commit.Build == nil {
		if resp.NotFoundError != nil {
			return "", errs.New("Commit not found: %s", resp.NotFoundError.Message)
		}
		return "", errs.New("Commit does not contain build")
	}

	if resp.Commit.Build.PlanningError != nil {
		var errs []string
		var isTransient bool
		for _, se := range resp.Commit.Build.SubErrors {
			errs = append(errs, se.Message)
			isTransient = se.IsTransient
		}
		return "", &BuildPlannerError{
			wrapped:          locale.NewInputError("err_buildplanner", resp.Commit.Build.Message),
			validationErrors: errs,
			isTransient:      isTransient,
		}
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

func (bp *BuildPlanner) GetBuildExpression(owner, project, commitID string) (*buildexpression.BuildExpression, error) {
	logging.Debug("GetBuildExpression")
	resp := &bpModel.BuildPlan{}
	err := bp.client.Run(request.BuildExpression(commitID), resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch build expression")
	}

	if resp.Commit == nil {
		return nil, errs.New("Staged commit is nil")
	}

	if resp.Commit.Expression == nil {
		return nil, errs.New("Commit does not contain expression")
	}

	expression, err := buildexpression.New(resp.Commit.Expression)
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse build expression")
	}

	return expression, nil
}
