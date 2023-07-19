package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/go-openapi/strfmt"
)

type Operation int

const (
	OperationAdded Operation = iota
	OperationRemoved
	OperationUpdated

	// BuildPlan statuses
	Planning  = "PLANNING"
	Planned   = "PLANNED"
	Building  = "BUILDING"
	Completed = "COMPLETED"

	// Artifact statuses
	ArtifactNotSubmitted      = "NOT_SUBMITTED"
	ArtifactBlocked           = "BLOCKED"
	ArtifactFailedPermanently = "FAILED_PERMANENTLY"
	ArtifactFailedTransiently = "FAILED_TRANSIENTLY"
	ArtifactReady             = "READY"
	ArtifactRunning           = "RUNNING"
	ArtifactSkipped           = "SKIPPED"
	ArtifactSucceeded         = "SUCCEEDED"

	// Types
	NotFound                 = "NotFound"
	BuildResultPlanningError = "PlanningError"

	// Tag types
	TagSource     = "src"
	TagDependency = "dep"
	TagBuilder    = "builder"
	TagOrphan     = "orphans"

	// BuildLogID types
	BuildLogRecipeID = "RECIPE_ID"
	BuildRequestID   = "BUILD_REQUEST_ID"

	ComparatorEQ  string = "eq"
	ComparatorGT         = "gt"
	ComparatorGTE        = "gte"
	ComparatorLT         = "lt"
	ComparatorLTE        = "lte"
	ComparatorNE         = "ne"

	VersionRequirementComparatorKey = "comparator"
	VersionRequirementVersionKey    = "version"

	XArtifactMimeType            = "application/x.artifact"
	XActiveStateArtifactMimeType = "application/x-activestate-artifacts"
	XCamelInstallerMimeType      = "application/x-camel-installer"
	XGozipInstallerMimeType      = "application/x-gozip-installer"
)

func IsStateToolArtifact(mimeType string) bool {
	return mimeType == XArtifactMimeType ||
		mimeType == XActiveStateArtifactMimeType ||
		mimeType == XCamelInstallerMimeType
}

func (o Operation) String() string {
	switch o {
	case OperationAdded:
		return "added"
	case OperationRemoved:
		return "removed"
	case OperationUpdated:
		return "updated"
	default:
		return "unknown"
	}
}

type BuildPlannerError struct {
	ValidationErrors []string
	IsTransient      bool
}

// InputError returns true as we want to treat all build planner errors as input errors
// and not report them to Rollbar. We defer the responsibility of logging these errors
// to the maintainers of the build planner.
func (e *BuildPlannerError) InputError() bool {
	return true
}

func (e *BuildPlannerError) Error() string {
	// Append last five lines to error message
	offset := 0
	numLines := len(e.ValidationErrors)
	if numLines > 5 {
		offset = numLines - 5
	}

	errorLines := strings.Join(e.ValidationErrors[offset:], "\n")
	// Crop at 500 characters to reduce noisy output further
	if len(errorLines) > 500 {
		offset = len(errorLines) - 499
		errorLines = fmt.Sprintf("…%s", errorLines[offset:])
	}
	isCropped := offset > 0
	croppedMessage := ""
	if isCropped {
		croppedMessage = locale.Tl("buildplan_err_cropped_intro", "These are the last lines of the error message:")
	}

	var err error
	err = locale.NewError("solver_err", "", croppedMessage, errorLines)
	if e.IsTransient {
		err = errs.AddTips(err, locale.Tr("transient_solver_tip"))
	}
	return err.Error()
}

// BuildPlan is the top level object returned by the build planner. It contains
// the commit and build.
type BuildPlan interface {
	Build() (*Build, error)
	CommitID() (strfmt.UUID, error)
}

func NewBuildPlanResponse(owner, project string) BuildPlan {
	if owner != "" && project != "" {
		return &BuildPlanByProject{}
	}
	return &BuildPlanByCommit{}
}

type BuildPlanByProject struct {
	Project *Project `json:"project"`
	*Error
}

func (b *BuildPlanByProject) Build() (*Build, error) {
	if b.Project == nil {
		return nil, errs.New("BuildPlanByProject.Build: Project is nil")
	}

	if b.Project.Error != nil {
		if b.Project.Error.Message != "" {
			return nil, errs.New("BuildPlanByProject.Build: Could not get build, API returned project error message: %s", b.Project.Message)
		}
		return nil, errs.New("BuildPlanByProject.Build: Could not retrieve project")
	}

	if b.Project.Commit == nil {
		return nil, errs.New("BuildPlanByProject.Build: Commit is nil")
	}

	if b.Project.Commit.Error != nil {
		if b.Project.Commit.Error.Message != "" {
			return nil, errs.New("Could not get build, API returned commit error message: %s", b.Project.Commit.Message)
		}
		return nil, errs.New("BuildPlanByProject.Build: Could not retrieve commit")
	}

	if b.Project.Commit.Type == NotFound {
		return nil, locale.NewError("err_buildplanner_commit_not_found", "Build plan does not contain commit")
	}

	if b.Project.Commit.Build == nil {
		return nil, errs.New("BuildPlanByProject.Build: Commit does not contain build")
	}

	if b.Project.Commit.Build.PlanningError != nil {
		var errs []string
		var isTransient bool
		for _, se := range b.Project.Commit.Build.SubErrors {
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
		return nil, &BuildPlannerError{
			ValidationErrors: errs,
			IsTransient:      isTransient,
		}
	}

	return b.Project.Commit.Build, nil
}

func (b *BuildPlanByProject) CommitID() (strfmt.UUID, error) {
	if b.Project == nil {
		return "", errs.New("BuildPlanByProject.CommitID: Project is nil")
	}

	if b.Project.Error != nil {
		if b.Project.Error.Message != "" {
			return "", errs.New("BuildPlanByProject.CommitID: Could not get commit ID, API returned project error message: %s", b.Project.Message)
		}
		return "", errs.New("BuildPlanByProject.CommitID: Could not retrieve project")
	}

	if b.Project.Commit == nil {
		return "", errs.New("BuildPlanByProject.CommitID: Commit is nil")
	}

	if b.Project.Commit.Error != nil {
		if b.Project.Commit.Error.Message != "" {
			return "", errs.New("BuildPlanByProject.CommitID: Could not get commit ID. API returned commit error message: %s", b.Project.Commit.Message)
		}
		return "", errs.New("BuildPlanByProject.CommitID: Could not retrieve commit")
	}

	return b.Project.Commit.CommitID, nil
}

type BuildPlanByCommit struct {
	Commit *Commit `json:"commit"`
	*Error
}

func (b *BuildPlanByCommit) Build() (*Build, error) {
	if b.Commit == nil {
		return nil, errs.New("BuildPlanByCommit.Build: Commit is nil")
	}

	if b.Commit.Error != nil {
		if b.Commit.Error.Message != "" {
			return nil, errs.New("BuildPlanByCommit.Build: Could not get build via commit ID, API returned commit error message: %s", b.Commit.Message)
		}
		return nil, errs.New("BuildPlanByCommit.Build: Could not retrieve commit")
	}

	if b.Commit.Type == NotFound {
		return nil, locale.NewError("err_buildplanner_commit_not_found", "Build plan does not contain commit")
	}

	if b.Commit.Build == nil {
		if b.Commit.Error != nil {
			return nil, errs.New("BuildPlanByCommit.Build: Commit not found: %s", b.Commit.Error.Message)
		}
		return nil, errs.New("BuildPlanByCommit.Build: Commit does not contain build")
	}

	if b.Commit.Build.PlanningError != nil {
		var errs []string
		var isTransient bool
		for _, se := range b.Commit.Build.SubErrors {
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
		return nil, &BuildPlannerError{
			ValidationErrors: errs,
			IsTransient:      isTransient,
		}
	}

	return b.Commit.Build, nil
}

func (b *BuildPlanByCommit) CommitID() (strfmt.UUID, error) {
	if b.Commit == nil {
		return "", errs.New("BuildPlanByCommit.CommitID: Commit is nil")
	}

	if b.Commit.Error != nil {
		if b.Commit.Error.Message != "" {
			return "", errs.New("BuildPlanByCommit.CommitID: Could not get commit ID, API returned commit error message: %s", b.Commit.Message)
		}
		return "", errs.New("BuildPlanByCommit.CommitID: Could not retrieve commit")
	}

	return b.Commit.CommitID, nil
}

type BuildExpression struct {
	Commit *Commit `json:"commit"`
	*Error
}

// PushCommitResult is the result of a push commit mutation.
// It contains the resulting commit from the operation and any errors.
// The resulting commit is pushed to the platform automatically.
type PushCommitResult struct {
	Commit *Commit `json:"pushCommit"`
	*Error
}

// StageCommitResult is the result of a stage commit mutation.
// It contains the resulting commit from the operation and any errors.
// The resulting commit is NOT pushed to the platform automatically.
type StageCommitResult struct {
	Commit *Commit `json:"stageCommit"`
	*Error
}

// Error contains an error message.
type Error struct {
	Message string `json:"message"`
}

// Project contains the commit and any errors.
type Project struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"commit"`
	*Error
}

// Commit contains the build and any errors.
type Commit struct {
	Type       string          `json:"__typename"`
	Expression json.RawMessage `json:"expr"`
	CommitID   strfmt.UUID     `json:"commitId"`
	Build      *Build          `json:"build"`
	*Error
}

// Build is a directed acyclic graph. It begins with a set of terminal nodes
// that resolve to artifacts via a set of steps.
// The expected format of a build plan is:
//
//	{
//	    "build": {
//	        "__typename": "BuildReady",
//	        "buildLogIds": [
//	            {
//	                "id": "1f717bf7-3573-5144-834b-75917dd8f60c",
//	                "type": "RECIPE_ID",
//	                "platformId": ""
//	            }
//	        ],
//	        "status": "READY",
//	        "terminals": [
//	            {
//	                "tag": "platform:78977bc8-0f32-519d-80f3-9043f059398c",
//	                "targetIDs": [
//	                    "311aacc7-a596-59c3-bbc9-cf2340721136",
//	                    "e02c6998-5357-5bc5-a785-6bd890a4af46"
//	                ]
//	            }
//	        ],
//	        "sources": [
//	            {
//	                "targetID": "6c91bc10-e8e2-50a6-8cca-ebd3f1e3f549",
//	                "name": "zlib",
//	                "namespace": "shared",
//	                "version": "1.2.13"
//	            },
//	            ...
//	        ],
//	        "steps": [
//	            {
//	                "targetID": "ab276a34-0b24-51b5-aacc-7323442f59ad",
//	                "inputs": [
//	                    {
//	                        "tag": "builder",
//	                        "targetIDs": [
//	                            "357d394b-6ce6-5385-be81-1754348fe5dd"
//	                        ]
//	                    },
//	                    {
//	                        "tag": "src",
//	                        "targetIDs": [
//	                            "bd5232a0-55de-52bd-ba29-1c58b9072232"
//	                        ]
//	                    },
//	                    {
//	                        "tag": "deps",
//	                        "targetIDs": []
//	                    }
//	                ],
//	                "outputs": [
//	                    "3ca4edd7-7746-55a1-a3cb-15b41b83ae52"
//	                ]
//	            },
//	            ...
//	        ],
//	        "artifacts": [
//	            {
//	                "__typename": "ArtifactSucceeded",
//	                "targetID": "7322308b-9789-50eb-b843-446cca78d855",
//	                "mimeType": "application/x-activestate-builder",
//	                "generatedBy": "8e5a488c-25b4-54b6-adfb-9d66d60f449f",
//	                "runtimeDependencies": [
//	                    "9a02d063-e3b6-5230-8cbe-f8769ced5a06",
//	                    "f9c838fc-e477-5f39-9cfc-3ffa804b4d53",
//	                    "b04ea3ed-9632-5e59-a571-201cfc225d36",
//	                    "2c64301a-9789-5cc3-b9b6-011bc7554268"
//	                ],
//	                "status": "SUCCEEDED",
//	                "logURL": "",
//	                "url": "s3://platform-sources/builder/0705c78c125b8b0f30e7fa6aeb30ac5f71c99511df40a6b62223be528f89385d/wheel-builder-lib.tar.gz",
//	                "checksum": "0705c78c125b8b0f30e7fa6aeb30ac5f71c99511df40a6b62223be528f89385d"
//	            },
//	            ...
//	        ]
//	    }
//	}
type Build struct {
	Type        string         `json:"__typename"`
	BuildPlanID strfmt.UUID    `json:"buildPlanID"`
	Status      string         `json:"status"`
	Terminals   []*NamedTarget `json:"terminals"`
	Artifacts   []*Artifact    `json:"artifacts"`
	Steps       []*Step        `json:"steps"`
	Sources     []*Source      `json:"sources"`
	BuildLogIDs []*BuildLogID  `json:"buildLogIds"`
	*PlanningError
}

// BuildLogID is the ID used to initiate a connection with the BuildLogStreamer.
type BuildLogID struct {
	ID         string      `json:"id"`
	PlatformID strfmt.UUID `json:"platformID"`
}

// NamedTarget is a special target used for terminals.
type NamedTarget struct {
	Tag     string        `json:"tag"`
	NodeIDs []strfmt.UUID `json:"nodeIds"`
}

// Artifact represents a downloadable artifact.
// This artifact may or may not be installable by the State Tool.
type Artifact struct {
	Type                string        `json:"__typename"`
	NodeID              strfmt.UUID   `json:"nodeId"`
	MimeType            string        `json:"mimeType"`
	GeneratedBy         strfmt.UUID   `json:"generatedBy"`
	RuntimeDependencies []strfmt.UUID `json:"runtimeDependencies"`
	Status              string        `json:"status"`
	URL                 string        `json:"url"`
	LogURL              string        `json:"logURL"`
	Checksum            string        `json:"checksum"`

	// Error fields
	Errors      []string `json:"errors"`
	Attempts    string   `json:"attempts"`
	NextAttempt string   `json:"nextAttempt"`
}

// Step represents a single step in the build plan.
// A step takes some input, processes it, and produces some output.
// This is usually a build step. The input represents a set of target
// IDs and the output are a set of artifact IDs.
type Step struct {
	StepID  strfmt.UUID    `json:"stepId"`
	Inputs  []*NamedTarget `json:"inputs"`
	Outputs []string       `json:"outputs"`
}

// Source represents the source of an artifact.
type Source struct {
	NodeID    strfmt.UUID `json:"nodeId"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Version   string      `json:"version"`
}

// PlanningError represents an error that occurred during planning.
type PlanningError struct {
	Message   string               `json:"message"`
	SubErrors []*BuildExprLocation `json:"subErrors"`
}

// BuildExprLocation represents a location in the build script where an error occurred.
type BuildExprLocation struct {
	Type             string                        `json:"__typename"`
	Path             string                        `json:"path"`
	Message          string                        `json:"message"`
	IsTransient      bool                          `json:"isTransient"`
	ValidationErrors []*SolverErrorValidationError `json:"validationErrors"`
	*RemediableSolveError
}

// SolverErrorValidationError represents a validation error that occurred during planning.
type SolverErrorValidationError struct {
	JSONPath string `json:"jsonPath"`
	Error    string `json:"error"`
}

// RemediableSolveError represents a solver error that can be remediated.
type RemediableSolveError struct {
	ErrorType         string                       `json:"errorType"`
	Remediations      []*SolverErrorRemediation    `json:"suggestedRemediations"`
	Requirements      []*Requirement               `json:"requirements"`
	Incompatibilities []*SolveErrorIncompatibility `json:"incompatibilities"`
}

type Requirement struct {
	Name               string               `json:"name"`
	Namespace          string               `json:"namespace"`
	VersionRequirement []VersionRequirement `json:"version_requirements,omitempty"`
}

type VersionRequirement map[string]string

// SolverErrorRemediation contains the recommeneded remediation for remediable error.
type SolverErrorRemediation struct {
	RemediationType string `json:"solveErrorRemediationType"`
	Command         string `json:"command"`
}

// SolverErrorIncompatibility represents a solver incompatibility error.
type SolveErrorIncompatibility struct {
	Type string `json:"type"`
	*SolveErrorPackageIncompatibility
	*SolveErrorPlatformIncompatibility
}

// SolveErrorPackageIncompatibility represents a package incompatibility error.
type SolveErrorPackageIncompatibility struct {
	Feature   string `json:"feature"`
	Namespace string `json:"namespace"`
}

// SolveErrorPlatformIncompatibility represents a platform incompatibility error.
type SolveErrorPlatformIncompatibility struct {
	PlatformID     string `json:"platformID"`
	PlatformKernel string `json:"platformKernel"`
}
