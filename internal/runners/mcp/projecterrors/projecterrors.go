package projecterrors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/graphql"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
)

type BuildNodesResponse struct {
	Commit struct {
		Build struct {
			Nodes []BuildNode `json:"nodes"`
		} `json:"build"`
	} `json:"commit"`
}

type BuildNode struct {
	Typename            string `json:"__typename"`
	Name                string `json:"name"`
	Namespace           string `json:"namespace"`
	Version             string `json:"version"`
	DisplayName         string `json:"displayName"`
	LogURL              string `json:"logURL"`
	Status              string `json:"status"`
	LastBuildFinishedAt string `json:"lastBuildFinishedAt"`
}

type RevisionResponse []struct {
	Versions []struct {
		Version   string `json:"version"`
		Revisions []struct {
			Revision          int    `json:"revision"`
			CreationTimestamp string `json:"creation_timestamp"`
		} `json:"revisions"`
	} `json:"versions"`
}

type FailedBuild struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	Namespace         string `json:"namespace"`
	BuildTimestamp    string `json:"build_timestamp"`
	LogURL            string `json:"logURL"`
	Fixed             bool   `json:"fixed"`
	IsDependencyError bool   `json:"is_dependency_error"`
}

type ProjectErrorsRunner struct {
	primer    *primer.Values
	namespace *project.Namespaced
}

func New(p *primer.Values, ns *project.Namespaced) *ProjectErrorsRunner {
	return &ProjectErrorsRunner{
		primer:    p,
		namespace: ns,
	}
}

func (runner *ProjectErrorsRunner) Run() error {
	gqlRunner := graphql.New(runner.primer.Auth(), api.ServiceBuildPlanner)
	request := &graphql.Request{
		QueryStr: `query($organization: String!, $project: String!) {
			project(organization: $organization, project: $project) {
				... on Project {
					commit {
						... on Commit {
							build {
								... on Build {
									nodes {
										... on Source {
											__typename, name, version, namespace
										}
										... on ArtifactPermanentlyFailed {
											__typename, displayName, logURL, status, lastBuildFinishedAt
										}
									}
								}
							}
						}
					}
				}
			}
		}`,
		QueryVars: map[string]interface{}{
			"organization": runner.namespace.Owner,
			"project":      runner.namespace.Project,
		},
	}

	response := BuildNodesResponse{}
	err := gqlRunner.Run(request, &response)
	if err != nil {
		return fmt.Errorf("error executing GraphQL query: %v", err)
	}

	// Process nodes to separate sources and failures
	sources := make(map[string]BuildNode)
	failures := make(map[string]BuildNode)

	for _, node := range response.Commit.Build.Nodes {
		switch node.Typename {
		case "Source":
			sources[node.Name] = node
		case "ArtifactPermanentlyFailed":
			failures[node.DisplayName] = node
		}
	}

	// Match failures with sources
	var failedBuilds []FailedBuild
	for failureName, failure := range failures {
		if source, exists := sources[failureName]; exists {
			fixed, err := checkNewerRevisionExists(runner.primer.Auth(), source.Name, source.Version, failure.LastBuildFinishedAt)
			if err != nil {
				return fmt.Errorf("error checking if ingredient is fixed: %v", err)
			}
			failedBuilds = append(failedBuilds, FailedBuild{
				Name:           source.Name,
				Version:        source.Version,
				Namespace:      source.Namespace,
				BuildTimestamp: failure.LastBuildFinishedAt,
				LogURL:         failure.LogURL,
				Fixed:          fixed,
			})
		}
	}

	if len(failedBuilds) > 0 {
		err := checkDependencyErrors(&failedBuilds)
		if err != nil {
			return fmt.Errorf("error checking dependency errors: %v", err)
		}
	}

	jsonBytes, err := json.Marshal(failedBuilds)
	if err != nil {
		return fmt.Errorf("error marshaling results: %w", err)
	}

	runner.primer.Output().Print(string(jsonBytes))

	return nil
}

func checkDependencyErrors(failedBuilds *[]FailedBuild) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	client := &http.Client{Timeout: 30 * time.Second}

	for i := range *failedBuilds {
		wg.Add(1)
		go func(build *FailedBuild) {
			defer wg.Done()
			depError, err := checkWasDependencyError(client, build.LogURL)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("error checking dependency error for %s: %v", build.Name, err))
				mu.Unlock()
				return
			}
			build.IsDependencyError = depError
		}(&(*failedBuilds)[i])
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errors)
	}

	return nil
}

func checkWasDependencyError(client *http.Client, logURL string) (bool, error) {
	resp, err := client.Get(logURL)
	if err != nil {
		return false, fmt.Errorf("error checking dependency error for %s: %v", logURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, logURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("error reading response body for %s: %v", logURL, err)
	}

	return strings.Contains(string(body), "ModuleNotFoundError"), nil
}

func checkNewerRevisionExists(auth *authentication.Auth, packageName, version, buildTimestamp string) (bool, error) {
	gqlRunner := graphql.New(auth, api.ServiceHasuraInventory)
	request := &graphql.Request{
		QueryStr: `
			query MyQuery($packageName: String!) {
				ingredient(where: {normalized_name: {_in: [$packageName]}}) {
					versions {
						version
						revisions {
							revision
							creation_timestamp
						}
					}
				}
			}
		`,
		QueryVars: map[string]interface{}{
			"packageName": packageName,
		},
	}

	response := RevisionResponse{}
	err := gqlRunner.Run(request, &response)
	if err != nil {
		return false, fmt.Errorf("error running the GraphQL request: %v", err)
	}

	if len(response) == 0 {
		return false, fmt.Errorf("no ingredient found for %s", packageName)
	}

	buildTime, err := time.Parse(time.RFC3339, buildTimestamp)
	if err != nil {
		return false, fmt.Errorf("error parsing build timestamp: %w", err)
	}

	for _, ingredient := range response {
		for _, v := range ingredient.Versions {
			if v.Version == version && len(v.Revisions) > 0 {
				var latestTime time.Time
				for _, revision := range v.Revisions {
					if revTime, err := time.Parse(time.RFC3339, revision.CreationTimestamp); err == nil {
						if revTime.After(latestTime) {
							latestTime = revTime
						}
					}
				}
				return buildTime.Before(latestTime), nil
			}
		}
	}

	return false, nil
}
