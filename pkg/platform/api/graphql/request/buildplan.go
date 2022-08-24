package request

func BuildPlanByCommitID(commitID string) *buildPlanByCommitID {
	return &buildPlanByCommitID{map[string]interface{}{
		"commitID": commitID,
	}}
}

type buildPlanByCommitID struct {
	vars map[string]interface{}
}

// TODO: Add error handling to this query
func (b *buildPlanByCommitID) Query() string {
	return `query ($commitID: String!) {
	project(project: "placeholder", organization: "placeholder") {
		... on Project {
			name
			description
			commit(vcsRef: $commitID) {
				... on Commit {
					parentId
					description
					build {
						__typename
						... on BuildReady {
							buildPlanID
							status
							terminals {
								__typename
								tag
								targetIDs
							}
							targets {
								... on Source {
									__typename
									targetID
									namespace
									name
									version
									revision
									ingredientID
									ingredientVersionID
								}
								... on Step {
									__typename
									targetID
									name
									inputs {
										__typename
										tag
										targetIDs
									}
									outputs
								}
								... on ArtifactSucceeded {
									__typename
									targetID
									mimeType
									status
									generatedBy
									runtimeDependencies
									status
									logURL
									url
									checksum
								}
							}
						}
					}
				}
			}
		}
	}
}
`
}

func (b *buildPlanByCommitID) Vars() map[string]interface{} {
	return b.vars
}
