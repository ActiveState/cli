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
      		__typename
				commit(vcsRef:$commitID) {
        			... on Commit {
          				__typename
          					build {
            					__typename
            					... on BuildReady {
									buildPlanID
									status
									terminals {
										tag
										targetIDs
									}
									targets {
										__typename
										... on Source {
											targetID
											name
											namespace
											version
										}
										... on Step {
											targetID
											inputs {
												tag
												targetIDs
											}
											outputs
										}
										... on ArtifactSucceeded {
											targetID
											mimeType
											generatedBy
											runtimeDependencies
											status
											logURL
											url
											checksum
										}
										... on ArtifactUnbuilt {
											targetID
											mimeType
											generatedBy
											runtimeDependencies
											status
										}
										... on ArtifactBuilding {
											targetID
											mimeType
											generatedBy
											runtimeDependencies
											status
										}
										... on ArtifactTransientlyFailed {
											targetID
											mimeType
											generatedBy
											runtimeDependencies
											status
											logURL
											errors
											attempts
											nextAttemptAt
										}
              						}
            					}
								... on BuildPlanned {
									buildPlanID
									status
									terminals {
										tag
										targetIDs
									}
								}
								... on BuildStarted {
									buildPlanID
									status
									terminals {
										tag
										targetIDs
									}
								}
								... on BuildPlanning {
									buildPlanID
									status
									terminals {
										tag
										targetIDs
									}
								}
								... on PlanningError {
									error
									subErrors {
										__typename
										... on GenericSolveError {
											path
											message
											isTransient
											validationErrors
										}
										... on RemediableSolveError {
											path
											message
											isTransient
											validationErrors
											suggestedRemediations {
												remediationType
												command
												parameters
											}
                						}
									}
            					}
          					}
        				}
					... on CommitNotFound {
						message
					}
				}
			}
		... on ProjectNotFound {
			__typename
			message
		}
	}
}
`
}

func (b *buildPlanByCommitID) Vars() map[string]interface{} {
	return b.vars
}
