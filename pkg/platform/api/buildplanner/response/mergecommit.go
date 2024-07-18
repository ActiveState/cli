package response

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
