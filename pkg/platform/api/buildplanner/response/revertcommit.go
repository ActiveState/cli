package response

import (
	"github.com/go-openapi/strfmt"
)

type RevertedCommit struct {
	Type           string      `json:"__typename"`
	Commit         *Commit     `json:"commit"`
	CommonAncestor strfmt.UUID `json:"commonAncestorID"`
	ConflictPaths  []string    `json:"conflictPaths"`
	*Error
	*ErrorWithSubErrors
}
