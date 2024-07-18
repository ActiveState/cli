package response

type BuildExpressionResponse struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"commit"`
	*Error
}
