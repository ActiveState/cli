package model

import (
	"net/http"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	platformModel "github.com/ActiveState/cli/pkg/platform/model"
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
	def    *Model
}

func NewBuildPlanner(auth *authentication.Auth) *BuildPlanner {
	bpURL := constants.APIBuildPlannerURL
	if url, ok := os.LookupEnv("_TEST_BUILDPLAN_URL"); ok {
		bpURL = url
	}

	return &BuildPlanner{
		auth:   auth,
		client: gqlclient.NewWithOpts(bpURL, 0, graphql.WithHTTPClient(&http.Client{})),
		def:    NewDefault(auth),
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
		if t.Tag == "orphans" {
			logging.Debug("Skipping")
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
		if string(platformID) == strings.TrimPrefix(t.Tag, "platform:") {
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
		res.Recipe, err = bp.def.ResolveRecipe(commitID, owner, project)
		if err != nil {
			return nil, locale.WrapError(err, "setup_build_resolve_recipe_err", "Could not resolve recipe for project {{.V0}}/{{.V1}}#{{.V2}}", owner, project, commitID.String())
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
