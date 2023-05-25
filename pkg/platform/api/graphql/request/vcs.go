package request

import (
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/go-openapi/strfmt"
)

func CheckpointByCommit(commitID strfmt.UUID) *checkpointByCommit {
	return &checkpointByCommit{vars: map[string]interface{}{
		"commit_id": commitID,
	}}
}

type checkpointByCommit struct {
	gqlclient.RequestBase
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
		vcs_commits_by_pk(commit_id: $commit_id) {
		  at_time
		}
	  }	  
	  `
}

func (p *checkpointByCommit) Vars() (map[string]interface{}, error) {
	return p.vars, nil
}
