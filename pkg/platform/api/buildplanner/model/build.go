package model

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/reqsimport"
	"github.com/ActiveState/graphql"
	"github.com/go-openapi/strfmt"
)

const (
	pollInterval       = 1 * time.Second
	pollTimeout        = 30 * time.Second
	buildStatusTimeout = 24 * time.Hour

	codeExtensionKey = "code"
)

func (c *client) Run(req gqlclient.Request, resp interface{}) error {
	logRequestVariables(req)
	return c.gqlClient.Run(req, resp)
}

func (bp *BuildPlanner) FetchCommitWithBuild(commitID strfmt.UUID, owner, project string, target *string) (*response.Commit, error) {
	logging.Debug("FetchBuildResult, commitID: %s, owner: %s, project: %s", commitID, owner, project)
	resp := &response.ProjectCommit{}
	err := bp.client.Run(request.ProjectCommit(commitID.String(), owner, project, target), resp)
	if err != nil {
		return nil, processBuildPlannerError(err, "failed to fetch build plan")
	}

	// The BuildPlanner will return a build plan with a status of
	// "planning" if the build plan is not ready yet. We need to
	// poll the BuildPlanner until the build is ready.
	if resp.Project.Commit.Build.Status == response.Planning {
		resp.Project.Commit.Build, err = bp.pollBuildPlanned(commitID.String(), owner, project, target)
		if err != nil {
			return nil, errs.Wrap(err, "failed to poll build plan")
		}
	}

	return resp.Project.Commit, nil
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
			return &response.BuildPlannerError{Err: locale.NewInputError("err_buildplanner_deprecated", "Encountered deprecation error: {{.V0}}", graphqlErr.Message)}
		}
	}
	return &response.BuildPlannerError{Err: locale.NewInputError("err_buildplanner", "{{.V0}}: Encountered unexpected error: {{.V1}}", fallbackMessage, bpErr.Error())}
}

var versionRe = regexp.MustCompile(`^\d+(\.\d+)*$`)

func isExactVersion(version string) bool {
	return versionRe.MatchString(version)
}

func isWildcardVersion(version string) bool {
	return strings.Contains(version, ".x") || strings.Contains(version, ".X")
}

func VersionStringToRequirements(version string) ([]response.VersionRequirement, error) {
	if isExactVersion(version) {
		return []response.VersionRequirement{{
			response.VersionRequirementComparatorKey: "eq",
			response.VersionRequirementVersionKey:    version,
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
		requirements := []response.VersionRequirement{}
		for _, change := range changeset {
			for _, constraint := range change.VersionConstraints {
				requirements = append(requirements, response.VersionRequirement{
					response.VersionRequirementComparatorKey: constraint.Comparator,
					response.VersionRequirementVersionKey:    constraint.Version,
				})
			}
		}
		return requirements, nil
	}

	// Construct version constraints to be >= given version, and < given version's last part + 1.
	// For example, given a version number of 3.10.x, constraints should be >= 3.10, < 3.11.
	// Given 2.x, constraints should be >= 2, < 3.
	requirements := []response.VersionRequirement{}
	parts := strings.Split(version, ".")
	for i, part := range parts {
		if part != "x" && part != "X" {
			continue
		}
		if i == 0 {
			return nil, locale.NewInputError("err_version_wildcard_start", "A version number cannot start with a wildcard")
		}
		requirements = append(requirements, response.VersionRequirement{
			response.VersionRequirementComparatorKey: response.ComparatorGTE,
			response.VersionRequirementVersionKey:    strings.Join(parts[:i], "."),
		})
		previousPart, err := strconv.Atoi(parts[i-1])
		if err != nil {
			return nil, locale.WrapInputError(err, "err_version_number_expected", "Version parts are expected to be numeric")
		}
		parts[i-1] = strconv.Itoa(previousPart + 1)
		requirements = append(requirements, response.VersionRequirement{
			response.VersionRequirementComparatorKey: response.ComparatorLT,
			response.VersionRequirementVersionKey:    strings.Join(parts[:i], "."),
		})
	}
	return requirements, nil
}

func (bp *BuildPlanner) BuildTarget(owner, project, commitID, target string) error {
	logging.Debug("BuildTarget, owner: %s, project: %s, commitID: %s, target: %s", owner, project, commitID, target)
	resp := &response.BuildTargetResult{}
	err := bp.client.Run(request.Evaluate(owner, project, commitID, target), resp)
	if err != nil {
		return processBuildPlannerError(err, "Failed to evaluate target")
	}

	if resp.Project == nil {
		return errs.New("Project is nil")
	}

	if response.IsErrorResponse(resp.Project.Type) {
		return response.ProcessProjectError(resp.Project, "Could not evaluate target")
	}

	if resp.Project.Commit == nil {
		return errs.New("Commit is nil")
	}

	if response.IsErrorResponse(resp.Project.Commit.Type) {
		return response.ProcessCommitError(resp.Project.Commit, "Could not process error response from evaluate target")
	}

	if resp.Project.Commit.Build == nil {
		return errs.New("Build is nil")
	}

	if response.IsErrorResponse(resp.Project.Commit.Build.Type) {
		return response.ProcessBuildError(resp.Project.Commit.Build, "Could not process error response from evaluate target")
	}

	return nil
}

// pollBuildPlanned polls the buildplan until it has passed the planning stage (ie. it's either planned or further along).
func (bp *BuildPlanner) pollBuildPlanned(commitID, owner, project string, target *string) (*response.Build, error) {
	resp := &response.ProjectCommit{}
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-ticker.C:
			err := bp.client.Run(request.ProjectCommit(commitID, owner, project, target), resp)
			if err != nil {
				return nil, processBuildPlannerError(err, "failed to fetch build plan")
			}

			if resp == nil {
				return nil, errs.New("Build plan response is nil")
			}

			build := resp.Project.Commit.Build

			if build.Status != response.Planning {
				return build, nil
			}
		case <-time.After(pollTimeout):
			return nil, locale.NewError("err_buildplanner_timeout", "Timed out waiting for build plan")
		}
	}
}

type ErrFailedArtifacts struct {
	Artifacts map[strfmt.UUID]*response.Artifact
}

func (e ErrFailedArtifacts) Error() string {
	return "ErrFailedArtifacts"
}

// WaitForBuild polls the build until it has passed the completed stage (ie. it's either successful or failed).
func (bp *BuildPlanner) WaitForBuild(commitID strfmt.UUID, owner, project string, target *string) error {
	failedArtifacts := map[strfmt.UUID]*response.Artifact{}
	resp := &response.ProjectCommit{}
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-ticker.C:
			err := bp.client.Run(request.ProjectCommit(commitID.String(), owner, project, target), resp)
			if err != nil {
				return processBuildPlannerError(err, "failed to fetch build plan")
			}

			if resp == nil {
				return errs.New("Build plan response is nil")
			}

			build := resp.Project.Commit.Build

			// If the build status is planning it may not have any artifacts yet.
			if build.Status == response.Planning {
				continue
			}

			// If all artifacts are completed then we are done.
			completed := true
			for _, artifact := range build.Artifacts {
				if artifact.Status == response.ArtifactNotSubmitted {
					continue
				}
				if artifact.Status != response.ArtifactSucceeded {
					completed = false
				}

				if artifact.Status == response.ArtifactFailedPermanently ||
					artifact.Status == response.ArtifactFailedTransiently {
					failedArtifacts[artifact.NodeID] = artifact
				}
			}

			if completed {
				return nil
			}

			// If the build status is completed then we are done.
			if build.Status == response.Completed {
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
