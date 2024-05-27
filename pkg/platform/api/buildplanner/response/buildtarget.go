package response

type BuildTargetResult struct {
	Build *BuildResponse `json:"buildCommitTarget"`
	*Error
	*NotFoundError
}
