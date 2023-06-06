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
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
	vcsModel "github.com/ActiveState/cli/pkg/platform/model"
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
	bpURL := api.GetServiceURL(api.ServiceBuildPlanner).String()
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

	// Check for errors in the response
	if resp.Project.Type == model.NotFound {
		return nil, locale.NewError("err_buildplanner_project_not_found", "Build plan does not contain project")
	}
	if resp.Project.Commit.Type == model.NotFound {
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

	if resp.Project.Commit.Build.Status == model.Planning {
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
		if t.Tag == model.TagOrphan {
			continue
		}
		bpPlatforms = append(bpPlatforms, strfmt.UUID(strings.TrimPrefix(t.Tag, "platform:")))
	}

	// Get the platform ID for the current platform
	platformID, err := platformModel.FilterCurrentPlatform(HostPlatform, bpPlatforms)
	if err != nil {
		return nil, locale.WrapError(err, "err_filter_current_platform")
	}

	// Filter the build terminals to only include the current platform
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
			break
		}
	}

	res := BuildResult{
		BuildEngine: buildEngine,
		Build:       resp.Project.Commit.Build,
		BuildReady:  resp.Project.Commit.Build.Status == model.Ready,
		CommitID:    strfmt.UUID(resp.Project.Commit.CommitID),
	}

	// We want to extract the recipe ID from the BuildLogIDs.
	// We do this because if the build is in progress we will need to reciepe ID to
	// initialize the build log streamer.
	// For camel builds the ID type will not be BuildLogRecipeID but this is okay
	// because the state tool does not display in progress information for camel builds.
	for _, id := range resp.Project.Commit.Build.BuildLogIDs {
		if id.Type == model.BuildLogRecipeID {
			if res.RecipeID != "" {
				return nil, errs.Wrap(err, "Build plan contains multiple recipe IDs")
			}
			res.RecipeID = strfmt.UUID(id.ID)
		}
	}

	return &res, nil
}

func (bp *BuildPlanner) pollBuildPlan(owner, project, commitID string) (*model.BuildPlan, error) {
	var resp *model.BuildPlan
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-ticker.C:
			err := bp.client.Run(request.BuildPlan(owner, project, commitID), resp)
			if err != nil {
				return nil, errs.Wrap(err, "failed to fetch build plan")
			}
			if resp.Project.Commit.Build.Status != model.Planning {
				return resp, nil
			}
		case <-time.After(pollTimeout):
			return nil, locale.NewError("err_buildplanner_timeout", "Timed out waiting for build plan")
		}
	}
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

type StageCommitParams struct {
	Owner        string
	Project      string
	ParentCommit string
	// Commits can have either an operation (e.g. installing a package)...
	PackageName      string
	PackageVersion   string
	PackageNamespace vcsModel.Namespace
	Operation        model.Operation
	// ... or a script (e.g. from pull).
	Script *model.BuildExpression
}

func (bp *BuildPlanner) StageCommit(params StageCommitParams) (strfmt.UUID, error) {
	script := params.Script
	if script == nil {
		var err error
		script, err = bp.GetBuildExpression(params.Owner, params.Project, params.ParentCommit)
		if err != nil {
			return "", errs.Wrap(err, "Failed to get build graph")
		}

		requirement := model.Requirement{
			Namespace: params.PackageNamespace.String(),
			Name:      params.PackageName,
		}

		if params.PackageVersion != "" {
			requirement.VersionRequirement = []model.VersionRequirement{{model.ComparatorEQ: params.PackageVersion}}
		}

		err = script.Update(params.Operation, requirement)
		if err != nil {
			return "", errs.Wrap(err, "Failed to update build graph")
		}
	}

	// With the updated build expression call the stage commit mutation
	request := request.StageCommit(params.Owner, params.Project, params.ParentCommit, script)
	resp := &model.StageCommitResult{}
	err := bp.client.Run(request, resp)
	if err != nil {
		return "", errs.Wrap(err, "failed to fetch build plan")
	}

	if resp.NotFoundError != nil {
		return "", errs.New("Commit not found: %s", resp.NotFoundError.Message)
	}

	if resp.Commit.Build.Status == model.Planning {
		buildResult, err := bp.FetchBuildResult(strfmt.UUID(resp.Commit.CommitID), params.Owner, params.Project)
		if err != nil {
			return "", errs.Wrap(err, "failed to fetch build result")
		}

		return buildResult.CommitID, nil
	}

	return strfmt.UUID(resp.Commit.CommitID), nil
}

func (bp *BuildPlanner) GetBuildExpression(owner, project, commitID string) (*model.BuildExpression, error) {
	resp := &model.BuildPlan{}
	err := bp.client.Run(request.BuildExpression(owner, project, commitID), resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch build graph")
	}

	if resp.Project.Type == model.NotFound {
		return nil, errs.New("Project not found: %s", resp.Project.Message)
	}
	if resp.Project.Commit.Type == model.NotFound {
		return nil, errs.New("Commit not found: %s", resp.Project.Commit.Message)
	}

	expression, err := model.NewBuildExpression(resp.Project.Commit.Script)
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse build expression")
	}

	return expression, nil
}
