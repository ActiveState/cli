package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"
)

type Operation int

type MergeStrategy string

const (
	OperationAdded Operation = iota
	OperationRemoved
	OperationUpdated

	// BuildPlan statuses
	Planning  = "PLANNING"
	Planned   = "PLANNED"
	Started   = "STARTED"
	Completed = "COMPLETED"

	// Artifact statuses
	ArtifactNotSubmitted      = "NOT_SUBMITTED"
	ArtifactBlocked           = "BLOCKED"
	ArtifactFailedPermanently = "FAILED_PERMANENTLY"
	ArtifactFailedTransiently = "FAILED_TRANSIENTLY"
	ArtifactReady             = "READY"
	ArtifactSkipped           = "SKIPPED"
	ArtifactStarted           = "STARTED"
	ArtifactSucceeded         = "SUCCEEDED"

	// Tag types
	TagSource     = "src"
	TagDependency = "deps"
	TagBuilder    = "builder"
	TagOrphan     = "orphans"

	// BuildLogID types
	BuildLogRecipeID = "RECIPE_ID"
	BuildRequestID   = "BUILD_REQUEST_ID"

	// Version Comparators
	ComparatorEQ  string = "eq"
	ComparatorGT         = "gt"
	ComparatorGTE        = "gte"
	ComparatorLT         = "lt"
	ComparatorLTE        = "lte"
	ComparatorNE         = "ne"

	// Version Requirement keys
	VersionRequirementComparatorKey = "comparator"
	VersionRequirementVersionKey    = "version"

	// MIME types
	XArtifactMimeType            = "application/x.artifact"
	XActiveStateArtifactMimeType = "application/x-activestate-artifacts"
	XCamelInstallerMimeType      = "application/x-camel-installer"
	XGozipInstallerMimeType      = "application/x-gozip-installer"
	XActiveStateBuilderMimeType  = "application/x-activestate-builder"

	// RevertCommit strategies
	RevertCommitStrategyForce   = "Force"
	RevertCommitStrategyDefault = "Default"

	// MergeCommit strategies
	MergeCommitStrategyRecursive                    MergeStrategy = "Recursive"
	MergeCommitStrategyRecursiveOverwriteOnConflict MergeStrategy = "RecursiveOverwriteOnConflict"
	MergeCommitStrategyRecursiveKeepOnConflict      MergeStrategy = "RecursiveKeepOnConflict"
	MergeCommitStrategyFastForward                  MergeStrategy = "FastForward"

	// Error types
	ErrorType                         = "Error"
	NotFoundErrorType                 = "NotFound"
	ParseErrorType                    = "ParseError"
	AlreadyExistsErrorType            = "AlreadyExists"
	NoChangeSinceLastCommitErrorType  = "NoChangeSinceLastCommit"
	HeadOnBranchMovedErrorType        = "HeadOnBranchMoved"
	ForbiddenErrorType                = "Forbidden"
	GenericSolveErrorType             = "GenericSolveError"
	RemediableSolveErrorType          = "RemediableSolveError"
	PlanningErrorType                 = "PlanningError"
	MergeConflictType                 = "MergeConflict"
	FastForwardErrorType              = "FastForwardError"
	NoCommonBaseFoundType             = "NoCommonBaseFound"
	ValidationErrorType               = "ValidationError"
	MergeConflictErrorType            = "MergeConflict"
	RevertConflictErrorType           = "RevertConflict"
	CommitNotInTargetHistoryErrorType = "CommitNotInTargetHistory"
	ComitHasNoParentErrorType         = "CommitHasNoParent"
)

func IsStateToolArtifact(mimeType string) bool {
	return mimeType == XArtifactMimeType ||
		mimeType == XActiveStateArtifactMimeType ||
		mimeType == XCamelInstallerMimeType
}

func IsSuccessArtifactStatus(status string) bool {
	return status == ArtifactSucceeded || status == ArtifactBlocked ||
		status == ArtifactStarted || status == ArtifactReady
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

func (o *Operation) Unmarshal(v string) error {
	switch v {
	case mono_models.CommitChangeEditableOperationAdded:
		*o = OperationAdded
	case mono_models.CommitChangeEditableOperationRemoved:
		*o = OperationRemoved
	case mono_models.CommitChangeEditableOperationUpdated:
		*o = OperationUpdated
	default:
		return errs.New("Unknown requirement operation: %s", v)
	}
	return nil
}

type BuildPlannerError struct {
	Err              error
	ValidationErrors []string
	IsTransient      bool
}

// InputError returns true as we want to treat all build planner errors as input errors
// and not report them to Rollbar. We defer the responsibility of logging these errors
// to the maintainers of the build planner.
func (e *BuildPlannerError) InputError() bool {
	return true
}

// UserError returns the error message to be displayed to the user.
// This function is added so that BuildPlannerErrors will be displayed
// to the user
func (e *BuildPlannerError) LocalizedError() string {
	return e.Error()
}

func (e *BuildPlannerError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

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

	if croppedMessage != "" {
		err = locale.NewError("buildplan_err_cropped", "", croppedMessage, errorLines)
	} else {
		err = locale.NewError("buildplan_err", "", errorLines)
	}

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
}

func (b *BuildPlanByProject) Build() (*Build, error) {
	if b.Project == nil {
		return nil, errs.New("BuildPlanByProject.Build: Project is nil")
	}

	if IsErrorResponse(b.Project.Type) {
		return nil, ProcessProjectError(b.Project, "Could not get build from project response")
	}

	if b.Project.Commit == nil {
		return nil, errs.New("BuildPlanByProject.Build: Commit is nil")
	}

	if IsErrorResponse(b.Project.Commit.Type) {
		return nil, ProcessCommitError(b.Project.Commit, "Could not get build from commit from project response")
	}

	if b.Project.Commit.Build == nil {
		return nil, errs.New("BuildPlanByProject.Build: Commit does not contain build")
	}

	if IsErrorResponse(b.Project.Commit.Build.Type) {
		return nil, ProcessBuildError(b.Project.Commit.Build, "Could not get build from project commit response")
	}

	return b.Project.Commit.Build, nil
}

func (b *BuildPlanByProject) CommitID() (strfmt.UUID, error) {
	if b.Project == nil {
		return "", errs.New("BuildPlanByProject.CommitID: Project is nil")
	}

	if IsErrorResponse(b.Project.Type) {
		return "", ProcessProjectError(b.Project, "Could not get commit ID from project response")
	}

	if b.Project.Commit == nil {
		return "", errs.New("BuildPlanByProject.CommitID: Commit is nil")
	}

	if IsErrorResponse(b.Project.Commit.Type) {
		return "", ProcessCommitError(b.Project.Commit, "Could not get commit ID from project commit response")
	}

	return b.Project.Commit.CommitID, nil
}

type BuildPlanByCommit struct {
	Commit *Commit `json:"commit"`
}

func (b *BuildPlanByCommit) Build() (*Build, error) {
	if b.Commit == nil {
		return nil, errs.New("BuildPlanByCommit.Build: Commit is nil")
	}

	if IsErrorResponse(b.Commit.Type) {
		return nil, ProcessCommitError(b.Commit, "Could not get build from commit response")
	}

	if b.Commit.Build == nil {
		return nil, errs.New("BuildPlanByCommit.Build: Commit does not contain build")
	}

	if IsErrorResponse(b.Commit.Build.Type) {
		return nil, ProcessBuildError(b.Commit.Build, "Could not get build from commit response")
	}

	return b.Commit.Build, nil
}

func (b *BuildPlanByCommit) CommitID() (strfmt.UUID, error) {
	if b.Commit == nil {
		return "", errs.New("BuildPlanByCommit.CommitID: Commit is nil")
	}

	if IsErrorResponse(b.Commit.Type) {
		return "", ProcessCommitError(b.Commit, "Could not get commit ID from commit response")
	}

	return b.Commit.CommitID, nil
}

func IsErrorResponse(errorType string) bool {
	return errorType == ErrorType ||
		errorType == NotFoundErrorType ||
		errorType == ParseErrorType ||
		errorType == AlreadyExistsErrorType ||
		errorType == NoChangeSinceLastCommitErrorType ||
		errorType == HeadOnBranchMovedErrorType ||
		errorType == ForbiddenErrorType ||
		errorType == RemediableSolveErrorType ||
		errorType == PlanningErrorType ||
		errorType == MergeConflictType ||
		errorType == FastForwardErrorType ||
		errorType == NoCommonBaseFoundType ||
		errorType == ValidationErrorType ||
		errorType == MergeConflictErrorType ||
		errorType == RevertConflictErrorType ||
		errorType == CommitNotInTargetHistoryErrorType ||
		errorType == ComitHasNoParentErrorType
}

type CommitError struct {
	Type                   string
	Message                string
	*locale.LocalizedError // for legacy, non-user-facing error usages
}

func ProcessCommitError(commit *Commit, fallbackMessage string) error {
	if commit.Error == nil {
		return errs.New(fallbackMessage)
	}

	switch commit.Type {
	case NotFoundErrorType:
		return &CommitError{
			commit.Type, commit.Message,
			locale.NewInputError("err_buildplanner_commit_not_found", "Could not find commit, received message: {{.V0}}", commit.Message),
		}
	case ParseErrorType:
		return &CommitError{
			commit.Type, commit.Message,
			locale.NewInputError("err_buildplanner_parse_error", "The platform failed to parse the build expression, received message: {{.V0}}. Path: {{.V1}}", commit.Message, commit.ParseError.Path),
		}
	case ForbiddenErrorType:
		return &CommitError{
			commit.Type, commit.Message,
			locale.NewInputError("err_buildplanner_forbidden", "Operation forbidden: {{.V0}}, received message: {{.V1}}", commit.Operation, commit.Message),
		}
	case HeadOnBranchMovedErrorType:
		return errs.Wrap(&CommitError{
			commit.Type, commit.Error.Message,
			locale.NewInputError("err_buildplanner_head_on_branch_moved"),
		}, "received message: "+commit.Error.Message)
	case NoChangeSinceLastCommitErrorType:
		return errs.Wrap(&CommitError{
			commit.Type, commit.Error.Message,
			locale.NewInputError("err_buildplanner_no_change_since_last_commit", "No new changes to commit."),
		}, commit.Error.Message)
	default:
		return errs.New(fallbackMessage)
	}
}

func ProcessBuildError(build *Build, fallbackMessage string) error {
	logging.Debug("ProcessBuildError: build.Type=%s", build.Type)
	if build.Type == PlanningErrorType {
		var errs []string
		var isTransient bool

		if build.Message != "" {
			errs = append(errs, build.Message)
		}

		for _, se := range build.SubErrors {
			if se.Type != RemediableSolveErrorType && se.Type != GenericSolveErrorType {
				continue
			}

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
		return &BuildPlannerError{
			ValidationErrors: errs,
			IsTransient:      isTransient,
		}
	} else if build.Error == nil {
		return errs.New(fallbackMessage)
	}

	return locale.NewInputError("err_buildplanner_build", "Encountered error processing build response")
}

func ProcessProjectError(project *Project, fallbackMessage string) error {
	if project.Type == NotFoundErrorType {
		return errs.AddTips(
			locale.NewInputError("err_buildplanner_project_not_found", "Unable to find project, received message: {{.V0}}", project.Message),
			locale.T("tip_private_project_auth"),
		)
	}

	return errs.New(fallbackMessage)
}

type RevertCommitError struct {
	Type    string
	Message string
}

func (m *RevertCommitError) Error() string { return m.Message }

func ProcessRevertCommitError(rcErr *revertedCommit, fallbackMessage string) error {
	if rcErr.Type != "" {
		return &RevertCommitError{rcErr.Type, rcErr.Message}
	}
	return errs.New(fallbackMessage)
}

type ProjectCreatedError struct {
	Type    string
	Message string
}

func (p *ProjectCreatedError) Error() string { return p.Message }

func ProcessProjectCreatedError(pcErr *projectCreated, fallbackMessage string) error {
	if pcErr.Error == nil {
		return errs.New(fallbackMessage)
	}

	return &ProjectCreatedError{pcErr.Type, pcErr.Message}
}

type BuildExpression struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"commit"`
	*Error
}

type MergedCommitError struct {
	Type    string
	Message string
}

func (m *MergedCommitError) Error() string { return m.Message }

func ProcessMergedCommitError(mcErr *mergedCommit, fallbackMessage string) error {
	if mcErr.Type != "" {
		return &MergedCommitError{mcErr.Type, mcErr.Message}
	}
	return errs.New(fallbackMessage)
}

// PushCommitResult is the result of a push commit mutation.
// It contains the resulting commit from the operation and any errors.
// The resulting commit is pushed to the platform automatically.
type PushCommitResult struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"pushCommit"`
	*Error
}

// StageCommitResult is the result of a stage commit mutation.
// It contains the resulting commit from the operation and any errors.
// The resulting commit is NOT pushed to the platform automatically.
type StageCommitResult struct {
	Commit *Commit `json:"stageCommit"`
}

type projectCreated struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"commit"`
	*Error
	*NotFoundError
	*ParseError
	*ForbiddenError
}

type CreateProjectResult struct {
	ProjectCreated *projectCreated `json:"createProject"`
}

type revertedCommit struct {
	Type           string      `json:"__typename"`
	Commit         *Commit     `json:"commit"`
	CommonAncestor strfmt.UUID `json:"commonAncestorID"`
	ConflictPaths  []string    `json:"conflictPaths"`
	*Error
}

type RevertCommitResult struct {
	RevertedCommit *revertedCommit `json:"revertCommit"`
}

type mergedCommit struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"commit"`
	*Error
	*MergeConflictError
	*MergeError
	*NotFoundError
	*ParseError
	*ForbiddenError
	*HeadOnBranchMovedError
	*NoChangeSinceLastCommitError
}

// MergeCommitResult is the result of a merge commit mutation.
// The resulting commit is only pushed to the platform automatically if the target ref was a named
// branch and the merge strategy was FastForward.
type MergeCommitResult struct {
	MergedCommit *mergedCommit `json:"mergeCommit"`
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
	*ParseError
	*ForbiddenError
	*HeadOnBranchMovedError
	*NoChangeSinceLastCommitError
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
	*Error
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
	DisplayName         string        `json:"displayName"`
	MimeType            string        `json:"mimeType"`
	GeneratedBy         strfmt.UUID   `json:"generatedBy"`
	RuntimeDependencies []strfmt.UUID `json:"runtimeDependencies"`
	Status              string        `json:"status"`
	URL                 string        `json:"url"`
	LogURL              string        `json:"logURL"`
	Checksum            string        `json:"checksum"`

	// Error fields
	Errors      []string `json:"errors"`
	Attempts    float64  `json:"attempts"`
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

// NotFoundError represents an error that occurred because a resource was not found.
type NotFoundError struct {
	Type                  string `json:"type"`
	Resource              string `json:"resource"`
	MayNeedAuthentication bool   `json:"mayNeedAuthentication"`
}

// PlanningError represents an error that occurred during planning.
type PlanningError struct {
	SubErrors []*BuildExprLocation `json:"subErrors"`
}

// ParseError is an error that occurred while parsing the build expression.
type ParseError struct {
	Path string `json:"path"`
}

type ForbiddenError struct {
	Operation string `json:"operation"`
}

// HeadOnBranchMovedError represents an error that occurred because the head on
// a remote branch has moved.
type HeadOnBranchMovedError struct {
	HeadBranchID strfmt.UUID `json:"branchId"`
}

// NoChangeSinceLastCommitError represents an error that occurred because there
// were no changes since the last commit.
type NoChangeSinceLastCommitError struct {
	NoChangeCommitID strfmt.UUID `json:"commitId"`
}

// MergeConflictError represents an error that occurred because of a merge conflict.
type MergeConflictError struct {
	CommonAncestorID strfmt.UUID `json:"commonAncestorId"`
	ConflictPaths    []string    `json:"conflictPaths"`
}

// MergeError represents two different errors in the BuildPlanner's graphQL
// schema with the same fields. Those errors being: FastForwardError and
// NoCommonBaseFound. Inspect the Type field to determine which error it is.
type MergeError struct {
	TargetVCSRef strfmt.UUID `json:"targetVcsRef"`
	OtherVCSRef  strfmt.UUID `json:"otherVcsRef"`
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
	Revision           *int                 `json:"revision,omitempty"`
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
