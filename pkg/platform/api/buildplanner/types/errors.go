package types

const (
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
	TargetNotFoundErrorType           = "TargetNotFound"
)
