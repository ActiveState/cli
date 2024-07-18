package response

import (
	"github.com/go-openapi/strfmt"
)

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
