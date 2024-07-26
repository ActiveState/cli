package response

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/go-openapi/strfmt"
)

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
	case types.NotFoundErrorType:
		return &CommitError{
			commit.Type, commit.Message,
			locale.NewInputError("err_buildplanner_commit_not_found", "Could not find commit. Received message: {{.V0}}", commit.Message),
		}
	case types.ParseErrorType:
		return &CommitError{
			commit.Type, commit.Message,
			locale.NewInputError("err_buildplanner_parse_error", "The platform failed to parse the build expression. Received message: {{.V0}}. Path: {{.V1}}", commit.Message, commit.ParseError.Path),
		}
	case types.ValidationErrorType:
		var subErrorMessages []string
		for _, e := range commit.SubErrors {
			subErrorMessages = append(subErrorMessages, e.Message)
		}
		if len(subErrorMessages) > 0 {
			return &CommitError{
				commit.Type, commit.Message,
				locale.NewInputError("err_buildplanner_validation_error_sub_messages", "The platform encountered a validation error. Received message: {{.V0}}, with sub errors: {{.V1}}", commit.Message, strings.Join(subErrorMessages, ", ")),
			}
		}
		return &CommitError{
			commit.Type, commit.Message,
			locale.NewInputError("err_buildplanner_validation_error", "The platform encountered a validation error. Received message: {{.V0}}", commit.Message),
		}
	case types.ForbiddenErrorType:
		return &CommitError{
			commit.Type, commit.Message,
			locale.NewInputError("err_buildplanner_forbidden", "Operation forbidden: {{.V0}}. Received message: {{.V1}}", commit.Operation, commit.Message),
		}
	case types.HeadOnBranchMovedErrorType:
		return errs.Wrap(&CommitError{
			commit.Type, commit.Error.Message,
			locale.NewInputError("err_buildplanner_head_on_branch_moved"),
		}, "received message: "+commit.Error.Message)
	case types.NoChangeSinceLastCommitErrorType:
		return errs.Wrap(&CommitError{
			commit.Type, commit.Error.Message,
			locale.NewInputError("err_buildplanner_no_change_since_last_commit", "No new changes to commit."),
		}, commit.Error.Message)
	default:
		return errs.New(fallbackMessage)
	}
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
