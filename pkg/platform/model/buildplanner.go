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
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/ActiveState/graphql"
	"github.com/go-openapi/strfmt"
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
	BuildExpression     *buildexpression.BuildExpression
}

func (b *BuildResult) OrderedArtifacts() []artifact.ArtifactID {
	res := make([]artifact.ArtifactID, 0, len(b.Build.Artifacts))
	for _, a := range b.Build.Artifacts {
		res = append(res, a.NodeID)
	}
	return res
}

type BuildPlanner struct {
	auth   *authentication.Auth
	client *gqlclient.Client
}

func NewBuildPlannerModel(auth *authentication.Auth) *BuildPlanner {
	bpURL := api.GetServiceURL(api.ServiceBuildPlanner).String()
	logging.Debug("Using build planner at: %s", bpURL)

	client := gqlclient.NewWithOpts(bpURL, 0, graphql.WithHTTPClient(&http.Client{}))

	if auth != nil && auth.Authenticated() {
		client.SetTokenProvider(auth)
	}

	return &BuildPlanner{
		auth:   auth,
		client: client,
	}
}

func (bp *BuildPlanner) FetchBuildResult(commitID strfmt.UUID, owner, project string) (*BuildResult, error) {
	logging.Debug("FetchBuildResult, commitID: %s, owner: %s, project: %s", commitID, owner, project)
	resp := bpModel.NewBuildPlanResponse(owner, project)
	err := bp.client.Run(request.BuildPlan(commitID.String(), owner, project), resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch build plan")
	}

	build, err := resp.Build()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get build from response")
	}

	// The BuildPlanner will return a build plan with a status of
	// "planning" if the build plan is not ready yet. We need to
	// poll the BuildPlanner until the build is ready.
	if build.Status == bpModel.Planning {
		build, err = bp.pollBuildPlan(commitID.String(), owner, project)
		if err != nil {
			return nil, errs.Wrap(err, "failed to poll build plan")
		}
	}

	// The type aliasing in the query populates the
	// response with emtpy targets that we should remove
	removeEmptyTargets(build)

	// Extract the available platforms from the build plan
	var bpPlatforms []strfmt.UUID
	for _, t := range build.Terminals {
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
	for _, t := range build.Terminals {
		if platformID.String() == strings.TrimPrefix(t.Tag, "platform:") {
			filteredTerminals = append(filteredTerminals, t)
		}
	}
	build.Terminals = filteredTerminals

	buildEngine := Alternative
	for _, s := range build.Sources {
		if s.Namespace == "builder" && s.Name == "camel" {
			buildEngine = Camel
			break
		}
	}

	id, err := resp.CommitID()
	if err != nil {
		return nil, errs.Wrap(err, "Response does not contain commitID")
	}

	expr, err := bp.GetBuildExpression(owner, project, commitID.String())
	if err != nil {
		return nil, errs.Wrap(err, "Failed to get build expression")
	}

	res := BuildResult{
		BuildEngine:     buildEngine,
		Build:           build,
		BuildReady:      build.Status == bpModel.Completed,
		CommitID:        id,
		BuildExpression: expr,
	}

	// We want to extract the recipe ID from the BuildLogIDs.
	// We do this because if the build is in progress we will need to reciepe ID to
	// initialize the build log streamer.
	// This information will only be populated if the build is an alternate build.
	// This is specified in the build planner queries.
	for _, id := range build.BuildLogIDs {
		if res.RecipeID != "" {
			return nil, errs.Wrap(err, "Build plan contains multiple recipe IDs")
		}
		res.RecipeID = strfmt.UUID(id.ID)
		break
	}

	return &res, nil
}

func (bp *BuildPlanner) pollBuildPlan(commitID, owner, project string) (*bpModel.Build, error) {
	resp := model.NewBuildPlanResponse(owner, project)
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-ticker.C:
			err := bp.client.Run(request.BuildPlan(commitID, owner, project), resp)
			if err != nil {
				return nil, errs.Wrap(err, "failed to fetch build plan")
			}

			if resp == nil {
				return nil, errs.New("Build plan response is nil")
			}

			build, err := resp.Build()
			if err != nil {
				return nil, errs.Wrap(err, "Could not get build from response")
			}

			if build.Status != bpModel.Planning {
				return build, nil
			}
		case <-time.After(pollTimeout):
			return nil, locale.NewError("err_buildplanner_timeout", "Timed out waiting for build plan")
		}
	}
}

func removeEmptyTargets(bp *bpModel.Build) {
	var steps []*bpModel.Step
	for _, step := range bp.Steps {
		if step.StepID == "" {
			continue
		}
		steps = append(steps, step)
	}

	var sources []*bpModel.Source
	for _, source := range bp.Sources {
		if source.NodeID == "" {
			continue
		}
		sources = append(sources, source)
	}

	var artifacts []*bpModel.Artifact
	for _, artifact := range bp.Artifacts {
		if artifact.NodeID == "" {
			continue
		}
		artifacts = append(artifacts, artifact)
	}

	bp.Steps = steps
	bp.Sources = sources
	bp.Artifacts = artifacts
}

type StageCommitParams struct {
	Owner        string
	Project      string
	ParentCommit string
	// Commits can have either an operation (e.g. installing a package)...
	RequirementName      string
	RequirementVersion   string
	RequirementNamespace Namespace
	Operation            bpModel.Operation
	TimeStamp            *strfmt.DateTime
	// ... or commits can have an expression (e.g. from pull). When pulling an expression, we do not
	// compute its changes into a series of above operations. Instead, we just pass the new
	// expression directly.
	Expression *buildexpression.BuildExpression
}

func (bp *BuildPlanner) StageCommit(params StageCommitParams) (strfmt.UUID, error) {
	logging.Debug("StageCommit, params: %+v", params)
	expression := params.Expression
	if expression == nil {
		var err error
		expression, err = bp.GetBuildExpression(params.Owner, params.Project, params.ParentCommit)
		if err != nil {
			return "", errs.Wrap(err, "Failed to get build expression")
		}

		if params.RequirementNamespace.Type() == NamespacePlatform {
			err = expression.UpdatePlatform(params.Operation, strfmt.UUID(params.RequirementName))
			if err != nil {
				return "", errs.Wrap(err, "Failed to update build expression with platform")
			}
		} else {
			requirement := bpModel.Requirement{
				Namespace: params.RequirementNamespace.String(),
				Name:      params.RequirementName,
			}

			if params.RequirementVersion != "" {
				requirement.VersionRequirement = []bpModel.VersionRequirement{{bpModel.VersionRequirementComparatorKey: bpModel.ComparatorEQ, bpModel.VersionRequirementVersionKey: params.RequirementVersion}}
			}

			err = expression.UpdateRequirement(params.Operation, requirement)
			if err != nil {
				return "", errs.Wrap(err, "Failed to update build expression with requirement")
			}
		}

		err = expression.UpdateTimestamp(*params.TimeStamp)
		if err != nil {
			return "", errs.Wrap(err, "Failed to update build expression with timestamp")
		}
	}

	// With the updated build expression call the stage commit mutation
	request := request.StageCommit(params.Owner, params.Project, params.ParentCommit, expression)
	resp := &bpModel.StageCommitResult{}
	err := bp.client.Run(request, resp)
	if err != nil {
		return "", locale.WrapError(err, "err_buildplanner_stage_commit", "Failed to stage commit, error: {{.V0}}", err.Error())
	}

	if resp.Commit == nil {
		return "", errs.New("Staged commit is nil")
	}

	if resp.Commit.Build == nil {
		if resp.Error != nil {
			return "", errs.New(resp.Error.Message)
		}
		return "", errs.New("Commit does not contain build")
	}

	if resp.Commit.Build.PlanningError != nil {
		var errs []string
		var isTransient bool
		for _, se := range resp.Commit.Build.SubErrors {
			if se.Message != "" {
				errs = append(errs, se.Message)
				isTransient = se.IsTransient
			}
			for _, ve := range se.ValidationErrors {
				if ve.Error != "" {
					errs = append(errs, ve.Error)
				}
			}
		}
		return "", &bpModel.BuildPlannerError{
			ValidationErrors: errs,
			IsTransient:      isTransient,
		}
	}

	if resp.Commit.Build.Status == bpModel.Planning {
		buildResult, err := bp.FetchBuildResult(strfmt.UUID(resp.Commit.CommitID), params.Owner, params.Project)
		if err != nil {
			return "", errs.Wrap(err, "failed to fetch build result")
		}

		return buildResult.CommitID, nil
	}

	return resp.Commit.CommitID, nil
}

func (bp *BuildPlanner) GetBuildExpression(owner, project, commitID string) (*buildexpression.BuildExpression, error) {
	logging.Debug("GetBuildExpression, owner: %s, project: %s, commitID: %s", owner, project, commitID)
	resp := &bpModel.BuildExpression{}
	err := bp.client.Run(request.BuildExpression(commitID), resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch build expression")
	}

	if resp.Commit == nil {
		return nil, errs.New("Commit is nil")
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
