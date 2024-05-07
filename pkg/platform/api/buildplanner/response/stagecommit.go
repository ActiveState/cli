package response

// StageCommitResult is the result of a stage commit mutation.
// It contains the resulting commit from the operation and any errors.
// The resulting commit is NOT pushed to the platform automatically.
type StageCommitResult struct {
	Commit *Commit `json:"stageCommit"`
}
