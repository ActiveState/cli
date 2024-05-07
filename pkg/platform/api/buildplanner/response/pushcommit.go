package response

// PushCommitResult is the result of a push commit mutation.
// It contains the resulting commit from the operation and any errors.
// The resulting commit is pushed to the platform automatically.
type PushCommitResult struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"pushCommit"`
	*Error
}
