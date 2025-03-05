package request

import (
	"github.com/go-openapi/strfmt"
)

func CheckpointByCommit(commitID strfmt.UUID) *checkpointByCommit {
	return &checkpointByCommit{vars: map[string]interface{}{
		"commit_id": commitID,
	}}
}

type checkpointByCommit struct {
	vars map[string]interface{}
}

func (p *checkpointByCommit) Query() string {
	return `query ($commit_id: uuid!) {
		vcs_checkpoints(where: {commit_id:{_eq: $commit_id}}) {
		  commit_id
		  namespace
		  requirement
		  version_constraint
		  constraint_json
		}
	  }`
}

func (p *checkpointByCommit) Vars() (map[string]interface{}, error) {
	return p.vars, nil
}

// New request type for commit details
func CommitByID(commitID strfmt.UUID) *commitByID {
	return &commitByID{vars: map[string]interface{}{
		"commit_id": commitID,
	}}
}

type commitByID struct {
	vars map[string]interface{}
}

func (p *commitByID) Query() string {
	return `query ($commit_id: uuid!) {
		vcs_commits_by_pk(commit_id: $commit_id) {
		  at_time
		}
	}`
}

func (p *commitByID) Vars() (map[string]interface{}, error) {
	return p.vars, nil
}
