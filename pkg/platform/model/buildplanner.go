package model

import (
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"runtime"
	"strconv"
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
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/reqsimport"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/ActiveState/graphql"
	"github.com/go-openapi/strfmt"
)

const (
	pollInterval       = 1 * time.Second
	pollTimeout        = 30 * time.Second
	buildStatusTimeout = 24 * time.Hour

	codeExtensionKey          = "code"
	clientDeprecationErrorKey = "CLIENT_DEPRECATION_ERROR"
)

// HostPlatform stores a reference to current platform
var HostPlatform string

type client struct {
	gqlClient *gqlclient.Client
}

func (c *client) Run(req gqlclient.Request, resp interface{}) error {
	logRequestVariables(req)
	return c.gqlClient.Run(req, resp)
}

func logRequestVariables(req gqlclient.Request) {
	if !strings.EqualFold(os.Getenv(constants.DebugServiceRequestsEnvVarName), "true") {
		return
	}

	vars, err := req.Vars()
	if err != nil {
		// Don't fail request because of this errors
		logging.Error("Failed to get request vars: %s", err)
		return
	}

	for _, v := range vars {
		if _, ok := v.(*buildexpression.BuildExpression); !ok {
			continue
		}

		beData, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			logging.Error("Failed to marshal build expression: %s", err)
			return
		}
		logging.Debug("Build Expression: %s", string(beData))
	}
}

type SolverError struct {
	wrapped          error
	validationErrors []string
	isTransient      bool
}

func (e *SolverError) Error() string {
	return "buildplan_error"
}

func (e *SolverError) Unwrap() error {
	return e.wrapped
}

func (e *SolverError) ValidationErrors() []string {
	return e.validationErrors
}

func (e *SolverError) IsTransient() bool {
	return e.isTransient
}

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
	BuildStatus         string
	BuildReady          bool
	BuildExpression     *buildexpression.BuildExpression
	AtTime              *strfmt.DateTime
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
	client *client
}

func NewBuildPlannerModel(auth *authentication.Auth) *BuildPlanner {
	bpURL := api.GetServiceURL(api.ServiceBuildPlanner).String()
	logging.Debug("Using build planner at: %s", bpURL)

	gqlClient := gqlclient.NewWithOpts(bpURL, 0, graphql.WithHTTPClient(api.NewHTTPClient()))

	if auth != nil && auth.Authenticated() {
		gqlClient.SetTokenProvider(auth)
	}

	return &BuildPlanner{
		auth: auth,
		client: &client{
			gqlClient: gqlClient,
		},
	}
}

func (bp *BuildPlanner) FetchBuildResult(commitID strfmt.UUID, owner, project string) (*BuildResult, *model.Commit, error) {
	logging.Debug("FetchBuildResult, commitID: %s, owner: %s, project: %s", commitID, owner, project)
	resp := bpModel.NewBuildPlanResponse(owner, project)
	err := bp.client.Run(request.BuildPlan(commitID.String(), owner, project), resp)
	if err != nil {
		return nil, nil, processBuildPlannerError(err, "failed to fetch build plan")
	}

	build, err := resp.Build()
	if err != nil {
		return nil, nil, errs.Wrap(err, "Could not get build from response")
	}

	// The BuildPlanner will return a build plan with a status of
	// "planning" if the build plan is not ready yet. We need to
	// poll the BuildPlanner until the build is ready.
	if build.Status == bpModel.Planning {
		build, err = bp.pollBuildPlan(commitID.String(), owner, project)
		if err != nil {
			return nil, nil, errs.Wrap(err, "failed to poll build plan")
		}
	}

	// The type aliasing in the query populates the
	// response with emtpy targets that we should remove
	removeEmptyTargets(build)

	buildEngine := Alternative
	for _, s := range build.Sources {
		if s.Namespace == "builder" && s.Name == "camel" {
			buildEngine = Camel
			break
		}
	}

	commit, err := resp.Commit()
	if err != nil {
		return nil, nil, errs.Wrap(err, "Response does not contain commitID")
	}

	expr, atTime, err := bp.GetBuildExpressionAndTime(commitID.String())
	if err != nil {
		return nil, nil, errs.Wrap(err, "Failed to get build expression and time")
	}

	res := BuildResult{
		BuildEngine:     buildEngine,
		Build:           build,
		BuildReady:      build.Status == bpModel.Completed,
		CommitID:        commit.CommitID,
		BuildExpression: expr,
		AtTime:          atTime,
		BuildStatus:     build.Status,
	}

	// We want to extract the recipe ID from the BuildLogIDs.
	// We do this because if the build is in progress we will need to reciepe ID to
	// initialize the build log streamer.
	// This information will only be populated if the build is an alternate build.
	// This is specified in the build planner queries.
	for _, id := range build.BuildLogIDs {
		if res.RecipeID != "" {
			return nil, nil, errs.Wrap(err, "Build plan contains multiple recipe IDs")
		}
		res.RecipeID = strfmt.UUID(id.ID)
	}

	return &res, commit, nil
}

func (bp *BuildPlanner) pollBuildPlan(commitID, owner, project string) (*bpModel.Build, error) {
	resp := model.NewBuildPlanResponse(owner, project)
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-ticker.C:
			err := bp.client.Run(request.BuildPlan(commitID, owner, project), resp)
			if err != nil {
				return nil, processBuildPlannerError(err, "failed to fetch build plan")
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

type StageCommitRequirement struct {
	Name      string
	Version   []bpModel.VersionRequirement
	Namespace Namespace
	Revision  *int
	Operation bpModel.Operation
}

type StageCommitParams struct {
	Owner        string
	Project      string
	ParentCommit string
	Description  string
	// Commits can have either requirements (e.g. installing a package)...
	Requirements []StageCommitRequirement
	// ... or commits can have an expression (e.g. from pull). When pulling an expression, we do not
	// compute its changes into a series of above operations. Instead, we just pass the new
	// expression directly.
	Expression *buildexpression.BuildExpression
	TimeStamp  *time.Time
}

func (bp *BuildPlanner) StageCommit(params StageCommitParams) (strfmt.UUID, error) {
	logging.Debug("StageCommit, params: %+v", params)
	expression := params.Expression
	if expression == nil {
		var err error
		expression, err = bp.GetBuildExpression(params.ParentCommit)
		if err != nil {
			return "", errs.Wrap(err, "Failed to get build expression")
		}

		var containsPackageOperation bool
		for _, req := range params.Requirements {
			if req.Namespace.Type() == NamespacePlatform {
				err = expression.UpdatePlatform(req.Operation, strfmt.UUID(req.Name))
				if err != nil {
					return "", errs.Wrap(err, "Failed to update build expression with platform")
				}
			} else {
				requirement := bpModel.Requirement{
					Namespace:          req.Namespace.String(),
					Name:               req.Name,
					VersionRequirement: req.Version,
					Revision:           req.Revision,
				}

				err = expression.UpdateRequirement(req.Operation, requirement)
				if err != nil {
					return "", errs.Wrap(err, "Failed to update build expression with requirement")
				}
				containsPackageOperation = true
			}
		}

		if containsPackageOperation {
			if err := expression.SetDefaultTimestamp(); err != nil {
				return "", errs.Wrap(err, "Failed to set default timestamp")
			}
		}

	}

	// With the updated build expression call the stage commit mutation
	request := request.StageCommit(params.Owner, params.Project, params.ParentCommit, params.Description, params.TimeStamp, expression)
	resp := &bpModel.StageCommitResult{}
	err := bp.client.Run(request, resp)
	if err != nil {
		return "", processBuildPlannerError(err, "failed to stage commit")
	}

	if resp.Commit == nil {
		return "", errs.New("Staged commit is nil")
	}

	if bpModel.IsErrorResponse(resp.Commit.Type) {
		return "", bpModel.ProcessCommitError(resp.Commit, "Could not process error response from stage commit")
	}

	if resp.Commit.CommitID == "" {
		return "", errs.New("Staged commit does not contain commitID")
	}

	return resp.Commit.CommitID, nil
}

func (bp *BuildPlanner) GetBuildExpression(commitID string) (*buildexpression.BuildExpression, error) {
	logging.Debug("GetBuildExpression, commitID: %s", commitID)
	resp := &bpModel.BuildExpression{}
	err := bp.client.Run(request.BuildExpression(commitID), resp)
	if err != nil {
		return nil, processBuildPlannerError(err, "failed to fetch build expression")
	}

	if resp.Commit == nil {
		return nil, errs.New("Commit is nil")
	}

	if bpModel.IsErrorResponse(resp.Commit.Type) {
		return nil, bpModel.ProcessCommitError(resp.Commit, "Could not get build expression from commit")
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

func (bp *BuildPlanner) GetBuildExpressionAndTime(commitID string) (*buildexpression.BuildExpression, *strfmt.DateTime, error) {
	logging.Debug("GetBuildExpressionAndTime, commitID: %s", commitID)
	resp := &bpModel.BuildExpression{}
	err := bp.client.Run(request.BuildExpression(commitID), resp)
	if err != nil {
		return nil, nil, processBuildPlannerError(err, "failed to fetch build expression")
	}

	if resp.Commit == nil {
		return nil, nil, errs.New("Commit is nil")
	}

	if bpModel.IsErrorResponse(resp.Commit.Type) {
		return nil, nil, bpModel.ProcessCommitError(resp.Commit, "Could not get build expression from commit")
	}

	if resp.Commit.Expression == nil {
		return nil, nil, errs.New("Commit does not contain expression")
	}

	expression, err := buildexpression.New(resp.Commit.Expression)
	if err != nil {
		return nil, nil, errs.Wrap(err, "failed to parse build expression")
	}

	return expression, &resp.Commit.AtTime, nil
}

// CreateProjectParams contains information for the project to create.
// When creating a project from scratch, the PlatformID, Language, Version, and Timestamp fields
// are used to create a buildexpression to use.
// When creating a project based off of another one, the Expr field is used (PlatformID, Language,
// Version, and Timestamp are ignored).
type CreateProjectParams struct {
	Owner       string
	Project     string
	PlatformID  strfmt.UUID
	Language    string
	Version     string
	Private     bool
	Description string
	Expr        *buildexpression.BuildExpression
}

func (bp *BuildPlanner) CreateProject(params *CreateProjectParams) (strfmt.UUID, error) {
	logging.Debug("CreateProject, owner: %s, project: %s, language: %s, version: %s", params.Owner, params.Project, params.Language, params.Version)

	expr := params.Expr
	if expr == nil {
		// Construct an initial buildexpression for the new project.
		var err error
		expr, err = buildexpression.NewEmpty()
		if err != nil {
			return "", errs.Wrap(err, "Unable to create initial buildexpression")
		}

		// Add the platform.
		if err := expr.UpdatePlatform(model.OperationAdded, params.PlatformID); err != nil {
			return "", errs.Wrap(err, "Unable to add platform")
		}

		// Create a requirement for the given language and version.
		versionRequirements, err := VersionStringToRequirements(params.Version)
		if err != nil {
			return "", errs.Wrap(err, "Unable to read version")
		}
		if err := expr.UpdateRequirement(model.OperationAdded, bpModel.Requirement{
			Name:               params.Language,
			Namespace:          "language", // TODO: make this a constant DX-1738
			VersionRequirement: versionRequirements,
		}); err != nil {
			return "", errs.Wrap(err, "Unable to add language requirement")
		}
	}

	// Create the project.
	request := request.CreateProject(params.Owner, params.Project, params.Private, expr, params.Description)
	resp := &bpModel.CreateProjectResult{}
	err := bp.client.Run(request, resp)
	if err != nil {
		return "", processBuildPlannerError(err, "Failed to create project")
	}

	if resp.ProjectCreated == nil {
		return "", errs.New("ProjectCreated is nil")
	}

	if bpModel.IsErrorResponse(resp.ProjectCreated.Type) {
		return "", bpModel.ProcessProjectCreatedError(resp.ProjectCreated, "Could not create project")
	}

	if resp.ProjectCreated.Commit == nil {
		return "", errs.New("ProjectCreated.Commit is nil")
	}

	return resp.ProjectCreated.Commit.CommitID, nil
}

func (bp *BuildPlanner) RevertCommit(organization, project, parentCommitID, commitID string) (strfmt.UUID, error) {
	logging.Debug("RevertCommit, organization: %s, project: %s, commitID: %s", organization, project, commitID)
	resp := &bpModel.RevertCommitResult{}
	err := bp.client.Run(request.RevertCommit(organization, project, parentCommitID, commitID), resp)
	if err != nil {
		return "", processBuildPlannerError(err, "Failed to revert commit")
	}

	if resp.RevertedCommit == nil {
		return "", errs.New("Revert commit response is nil")
	}

	if bpModel.IsErrorResponse(resp.RevertedCommit.Type) {
		return "", bpModel.ProcessRevertCommitError(resp.RevertedCommit, "Could not revert commit")
	}

	if resp.RevertedCommit.Commit == nil {
		return "", errs.New("Revert commit's commit is nil'")
	}

	if bpModel.IsErrorResponse(resp.RevertedCommit.Commit.Type) {
		return "", bpModel.ProcessCommitError(resp.RevertedCommit.Commit, "Could not process error response from revert commit")
	}

	if resp.RevertedCommit.Commit.CommitID == "" {
		return "", errs.New("Commit does not contain commitID")
	}

	return resp.RevertedCommit.Commit.CommitID, nil
}

type MergeCommitParams struct {
	Owner     string
	Project   string
	TargetRef string // the commit ID or branch name to merge into
	OtherRef  string // the commit ID or branch name to merge from
	Strategy  model.MergeStrategy
}

func (bp *BuildPlanner) MergeCommit(params *MergeCommitParams) (strfmt.UUID, error) {
	logging.Debug("MergeCommit, owner: %s, project: %s", params.Owner, params.Project)
	request := request.MergeCommit(params.Owner, params.Project, params.TargetRef, params.OtherRef, params.Strategy)
	resp := &bpModel.MergeCommitResult{}
	err := bp.client.Run(request, resp)
	if err != nil {
		return "", processBuildPlannerError(err, "Failed to merge commit")
	}

	if resp.MergedCommit == nil {
		return "", errs.New("MergedCommit is nil")
	}

	if bpModel.IsErrorResponse(resp.MergedCommit.Type) {
		return "", bpModel.ProcessMergedCommitError(resp.MergedCommit, "Could not merge commit")
	}

	if resp.MergedCommit.Commit == nil {
		return "", errs.New("Merge commit's commit is nil'")
	}

	if bpModel.IsErrorResponse(resp.MergedCommit.Commit.Type) {
		return "", bpModel.ProcessCommitError(resp.MergedCommit.Commit, "Could not process error response from merge commit")
	}

	return resp.MergedCommit.Commit.CommitID, nil
}

// processBuildPlannerError will check for special error types that should be
// handled differently. If no special error type is found, the fallback message
// will be used.
// It expects the errors field to be the top-level field in the response. This is
// different from special error types that are returned as part of the data field.
// Example:
//
//	{
//	  "errors": [
//	    {
//	      "message": "deprecation error",
//	      "locations": [
//	        {
//	          "line": 7,
//	          "column": 11
//	        }
//	      ],
//	      "path": [
//	        "project",
//	        "commit",
//	        "build"
//	      ],
//	      "extensions": {
//	        "code": "CLIENT_DEPRECATION_ERROR"
//	      }
//	    }
//	  ],
//	  "data": null
//	}
func processBuildPlannerError(bpErr error, fallbackMessage string) error {
	graphqlErr := &graphql.GraphErr{}
	if errors.As(bpErr, graphqlErr) {
		code, ok := graphqlErr.Extensions[codeExtensionKey].(string)
		if ok && code == clientDeprecationErrorKey {
			return &bpModel.BuildPlannerError{Err: locale.NewInputError("err_buildplanner_deprecated", "Encountered deprecation error: {{.V0}}", graphqlErr.Message)}
		}
	}
	return &bpModel.BuildPlannerError{Err: errs.Wrap(bpErr, fallbackMessage)}
}

var versionRe = regexp.MustCompile(`^\d+(\.\d+)*$`)

func isExactVersion(version string) bool {
	return versionRe.MatchString(version)
}

func isWildcardVersion(version string) bool {
	return strings.Contains(version, ".x") || strings.Contains(version, ".X")
}

func VersionStringToRequirements(version string) ([]bpModel.VersionRequirement, error) {
	if isExactVersion(version) {
		return []bpModel.VersionRequirement{{
			bpModel.VersionRequirementComparatorKey: "eq",
			bpModel.VersionRequirementVersionKey:    version,
		}}, nil
	}

	if !isWildcardVersion(version) {
		// Ask the Platform to translate a string like ">=1.2,<1.3" into a list of requirements.
		// Note that:
		// - The given requirement name does not matter; it is not looked up.
		changeset, err := reqsimport.Init().Changeset([]byte("name "+version), "")
		if err != nil {
			return nil, locale.WrapInputError(err, "err_invalid_version_string", "Invalid version string")
		}
		requirements := []bpModel.VersionRequirement{}
		for _, change := range changeset {
			for _, constraint := range change.VersionConstraints {
				requirements = append(requirements, bpModel.VersionRequirement{
					bpModel.VersionRequirementComparatorKey: constraint.Comparator,
					bpModel.VersionRequirementVersionKey:    constraint.Version,
				})
			}
		}
		return requirements, nil
	}

	// Construct version constraints to be >= given version, and < given version's last part + 1.
	// For example, given a version number of 3.10.x, constraints should be >= 3.10, < 3.11.
	// Given 2.x, constraints should be >= 2, < 3.
	requirements := []bpModel.VersionRequirement{}
	parts := strings.Split(version, ".")
	for i, part := range parts {
		if part != "x" && part != "X" {
			continue
		}
		if i == 0 {
			return nil, locale.NewInputError("err_version_wildcard_start", "A version number cannot start with a wildcard")
		}
		requirements = append(requirements, bpModel.VersionRequirement{
			bpModel.VersionRequirementComparatorKey: bpModel.ComparatorGTE,
			bpModel.VersionRequirementVersionKey:    strings.Join(parts[:i], "."),
		})
		previousPart, err := strconv.Atoi(parts[i-1])
		if err != nil {
			return nil, locale.WrapInputError(err, "err_version_number_expected", "Version parts are expected to be numeric")
		}
		parts[i-1] = strconv.Itoa(previousPart + 1)
		requirements = append(requirements, bpModel.VersionRequirement{
			bpModel.VersionRequirementComparatorKey: bpModel.ComparatorLT,
			bpModel.VersionRequirementVersionKey:    strings.Join(parts[:i], "."),
		})
	}
	return requirements, nil
}

func FilterCurrentPlatform(hostPlatform string, platforms []strfmt.UUID, cfg Configurable, auth *authentication.Auth) (strfmt.UUID, error) {
	platformIDs, err := filterPlatformIDs(hostPlatform, runtime.GOARCH, platforms, cfg, auth)
	if err != nil {
		return "", errs.Wrap(err, "filterPlatformIDs failed")
	}

	if len(platformIDs) == 0 {
		return "", locale.NewInputError("err_recipe_no_platform")
	} else if len(platformIDs) > 1 {
		logging.Debug("Received multiple platform IDs. Picking the first one: %s", platformIDs[0])
	}

	return platformIDs[0], nil
}

func (bp *BuildPlanner) BuildTarget(owner, project, commitID, target string) error {
	logging.Debug("BuildTarget, owner: %s, project: %s, commitID: %s, target: %s", owner, project, commitID, target)
	resp := &bpModel.BuildTargetResult{}
	err := bp.client.Run(request.Evaluate(owner, project, commitID, target), resp)
	if err != nil {
		return processBuildPlannerError(err, "Failed to evaluate target")
	}

	if resp.Project == nil {
		return errs.New("Project is nil")
	}

	if bpModel.IsErrorResponse(resp.Project.Type) {
		return bpModel.ProcessProjectError(resp.Project, "Could not evaluate target")
	}

	if resp.Project.Commit == nil {
		return errs.New("Commit is nil")
	}

	if bpModel.IsErrorResponse(resp.Project.Commit.Type) {
		return bpModel.ProcessCommitError(resp.Project.Commit, "Could not process error response from evaluate target")
	}

	if resp.Project.Commit.Build == nil {
		return errs.New("Build is nil")
	}

	if bpModel.IsErrorResponse(resp.Project.Commit.Build.Type) {
		return bpModel.ProcessBuildError(resp.Project.Commit.Build, "Could not process error response from evaluate target")
	}

	return nil
}

type ErrFailedArtifacts struct {
	Artifacts map[strfmt.UUID]*bpModel.Artifact
}

func (e ErrFailedArtifacts) Error() string {
	return "ErrFailedArtifacts"
}

func (bp *BuildPlanner) PollBuildStatus(commitID, owner, project, target string) error {
	failedArtifacts := map[strfmt.UUID]*model.Artifact{}
	resp := model.NewBuildPlanResponse(owner, project)
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-ticker.C:
			err := bp.client.Run(request.BuildPlanTarget(commitID, owner, project, target), resp)
			if err != nil {
				return processBuildPlannerError(err, "failed to fetch build plan")
			}

			if resp == nil {
				return errs.New("Build plan response is nil")
			}

			build, err := resp.Build()
			if err != nil {
				return errs.Wrap(err, "Could not get build from response")
			}

			// The type aliasing in the query populates the
			// response with emtpy targets that we should remove
			removeEmptyTargets(build)

			// If the build status is planning it may not have any artifacts yet.
			if build.Status == bpModel.Planning {
				continue
			}

			// If all artifacts are completed then we are done.
			completed := true
			for _, artifact := range build.Artifacts {
				if artifact.Status == bpModel.ArtifactNotSubmitted {
					continue
				}
				if artifact.Status != bpModel.ArtifactSucceeded {
					completed = false
				}

				if artifact.Status == bpModel.ArtifactFailedPermanently ||
					artifact.Status == bpModel.ArtifactFailedTransiently {
					failedArtifacts[artifact.NodeID] = artifact
				}
			}

			if completed {
				return nil
			}

			// If the build status is completed then we are done.
			if build.Status == bpModel.Completed {
				if len(failedArtifacts) != 0 {
					return ErrFailedArtifacts{failedArtifacts}
				}
				return nil
			}
		case <-time.After(buildStatusTimeout):
			return locale.NewError("err_buildplanner_timeout", "Timed out waiting for build plan")
		}
	}
}
