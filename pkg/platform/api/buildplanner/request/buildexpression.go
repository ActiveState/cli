package request

func BuildExpression(commitID string) *buildScriptByCommitID {
	return &buildScriptByCommitID{map[string]interface{}{
		"commitID": commitID,
	}}
}

type buildScriptByCommitID struct {
	vars map[string]interface{}
}

func (b *buildScriptByCommitID) Query() string {
	return `
query ($commitID: String!) {
  commit(commitId: $commitID) {
    ... on Commit {
      __typename
      script
    }
    ... on NotFound {
      __typename
      message
    }
  }
}
`
}

func (b *buildScriptByCommitID) Vars() map[string]interface{} {
	return b.vars
}