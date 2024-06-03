package request

type CommitRequest struct {
	owner    string
	project  string
	commitID string
}

func NewCommitRequest(owner, project, commitID string) *CommitRequest {
	return &CommitRequest{
		owner:    owner,
		project:  project,
		commitID: commitID,
	}
}

func (c *CommitRequest) Query() string {
	return `query($owner: String!, $project: String!, $commitID: String!)  {
		getCommit(owner: $owner, project: $project, id: $commitID) {
			atTime
			expression
			buildPlan
		}
	}`
}

func (c *CommitRequest) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{
		"owner":    c.owner,
		"project":  c.project,
		"commitID": c.commitID,
	}, nil
}
