package response

// MergedCommit is the result of a merge commit mutation.
// The resulting commit is only pushed to the platform automatically if the target ref was a named
// branch and the merge strategy was FastForward.
type MergedCommit struct {
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
