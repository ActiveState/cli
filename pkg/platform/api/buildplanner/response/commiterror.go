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
		var subErrorMessages []string
		for _, e := range commit.SubErrors {
			subErrorMessages = append(subErrorMessages, e.Message)
		}
		if len(subErrorMessages) > 0 {
			return &CommitError{
				commit.Type, commit.Message,
				locale.NewInputError("err_buildplanner_commit_parse_error_sub_messages", "The platform failed to parse the build expression. Received message: {{.V0}}, with sub errors: {{.V1}}", commit.Message, strings.Join(subErrorMessages, ", ")),
			}
		}
		return &CommitError{
			commit.Type, commit.Message,
			locale.NewInputError("err_buildplanner_commit_parse_error", "The platform failed to parse the build expression. Received message: {{.V0}}", commit.Message),
		}
	case types.ValidationErrorType:
		var subErrorMessages []string
		for _, e := range commit.SubErrors {
			subErrorMessages = append(subErrorMessages, e.Message)
		}
		if len(subErrorMessages) > 0 {
			return &CommitError{
				commit.Type, commit.Message,
				locale.NewInputError("err_buildplanner_commit_validation_error_sub_messages", "The platform encountered a validation error. Received message: {{.V0}}, with sub errors: {{.V1}}", commit.Message, strings.Join(subErrorMessages, ", ")),
			}
		}
		return &CommitError{
			commit.Type, commit.Message,
			locale.NewInputError("err_buildplanner_commit_validation_error", "The platform encountered a validation error. Received message: {{.V0}}", commit.Message),
		}
	case types.ForbiddenErrorType:
		return &CommitError{
			commit.Type, commit.Message,
			locale.NewInputError("err_buildplanner_forbidden", commit.Operation, commit.Message),
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
	*locale.LocalizedError
}

func (m *RevertCommitError) Error() string { return m.Message }

func ProcessRevertCommitError(rcErr *revertedCommit, fallbackMessage string) error {
	if rcErr.Error == nil {
		return errs.New(fallbackMessage)
	}

	switch rcErr.Type {
	case types.RevertConflictErrorType:
		return &RevertCommitError{
			rcErr.Type, rcErr.Message,
			locale.NewInputError("err_buildplanner_revert_conflict", "The revert operation could not be completed due to a conflict. Received message: {{.V0}}", rcErr.Message),
		}
	case types.CommitNotInTargetHistoryErrorType:
		return &RevertCommitError{
			rcErr.Type, rcErr.Message,
			locale.NewInputError("err_buildplanner_commit_revert_not_in_target_history", "The commit to revert is not in the target history. Received message: {{.V0}}", rcErr.Message),
		}
	case types.ComitHasNoParentErrorType:
		return &RevertCommitError{
			rcErr.Type, rcErr.Message,
			locale.NewInputError("err_buildplanner_commit_revert_has_no_parent", "The commit to revert has no parent. Received message: {{.V0}}", rcErr.Message),
		}
	case types.InvalidInputErrorType:
		return &RevertCommitError{
			rcErr.Type, rcErr.Message,
			locale.NewInputError("err_buildplanner_revert_invalid_input", "The input to the revert operation was invalid. Received message: {{.V0}}", rcErr.Message),
		}
	default:
		return errs.New(fallbackMessage)
	}
}

type MergedCommitError struct {
	Type    string
	Message string
	*locale.LocalizedError
}

func (m *MergedCommitError) Error() string { return m.Message }

func ProcessMergedCommitError(mcErr *mergedCommit, fallbackMessage string) error {
	if mcErr.Error == nil {
		return errs.New(fallbackMessage)
	}

	switch mcErr.Type {
	case types.MergeConflictErrorType:
		return &MergedCommitError{
			mcErr.Type, mcErr.Message,
			locale.NewInputError("err_buildplanner_merge_conflict", "The platform encountered a merge conflict. Received message: {{.V0}}", mcErr.Message),
		}
	case types.FastForwardErrorType:
		return &MergedCommitError{
			mcErr.Type, mcErr.Message,
			locale.NewInputError("err_buildplanner_merge_fast_forward_error", "The platform could not merge with the Fast Forward strategy. Received message: {{.V0}}", mcErr.Message),
		}
	case types.NoCommonBaseFoundErrorType:
		return &MergedCommitError{
			mcErr.Type, mcErr.Message,
			locale.NewInputError("err_buildplanner_merge_no_common_base_found", "The platform could not find a common base for the merge. Received message: {{.V0}}", mcErr.Message),
		}
	case types.InvalidInputErrorType:
		return &MergedCommitError{
			mcErr.Type, mcErr.Message,
			locale.NewInputError("err_buildplanner_merge_invalid_input", "The input to the merge commit mutation was invalid. Received message: {{.V0}}", mcErr.Message),
		}
	default:
		return errs.New(fallbackMessage)
	}
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
