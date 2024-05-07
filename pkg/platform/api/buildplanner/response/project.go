package response

// ProjectResponse contains the commit and any errors.
type ProjectResponse struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"commit"`
	*Error
}
