package types

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

type MergeStrategy string

type Operation int

const (
	OperationAdded Operation = iota
	OperationRemoved
	OperationUpdated
)

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

const (
	RevertCommitStrategyForce   = "Force"
	RevertCommitStrategyDefault = "Default"
)

const (
	MergeCommitStrategyRecursive                    MergeStrategy = "Recursive"
	MergeCommitStrategyRecursiveOverwriteOnConflict               = "RecursiveOverwriteOnConflict"
	MergeCommitStrategyRecursiveKeepOnConflict                    = "RecursiveKeepOnConflict"
	MergeCommitStrategyFastForward                                = "FastForward"
)
