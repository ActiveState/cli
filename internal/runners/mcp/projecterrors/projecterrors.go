package projecterrors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
)

type ProjectErrorsRunner struct {
	auth     *authentication.Auth
	output   output.Outputer
	svcModel *model.SvcModel
}

func New(p *primer.Values) *ProjectErrorsRunner {
	return &ProjectErrorsRunner{
		auth:     p.Auth(),
		output:   p.Output(),
		svcModel: p.SvcModel(),
	}
}

type Params struct {
	namespace *project.Namespaced
}

func NewParams(namespace *project.Namespaced) *Params {
	return &Params{
		namespace: namespace,
	}
}

type ArtifactOutput struct {
	Name                  string `json:"name"`
	Version               string `json:"version"`
	Namespace             string `json:"namespace"`
	IsBuildtimeDependency bool   `json:"isBuildtimeDependency"`
	IsRuntimeDependency   bool   `json:"isRuntimeDependency"`
	LogURL                string `json:"logURL"`
	SourceURI             string `json:"sourceURI"`
	WasFixed              bool   `json:"wasFixed"`
	IsDependencyError     bool   `json:"isDependencyError"`
}

func (runner *ProjectErrorsRunner) Run(params *Params) error {
	branch, err := model.DefaultBranchForProjectName(params.namespace.Owner, params.namespace.Project)
	if err != nil {
		return fmt.Errorf("error fetching default branch: %w", err)
	}

	bpm := buildplanner.NewBuildPlannerModel(runner.auth, runner.svcModel)
	commit, err := bpm.FetchCommitNoPoll(
		*branch.CommitID, params.namespace.Owner, params.namespace.Project, nil)
	if err != nil {
		return fmt.Errorf("error fetching commit: %w", err)
	}

	bp := commit.BuildPlan()
	failedArtifacts := bp.Artifacts(buildplan.FilterFailedArtifacts())

	// Check if artifacts have already been fixed by a newer revision.
	wasFixed, err := CheckDependencyFixes(runner.auth, failedArtifacts)
	if err != nil {
		return fmt.Errorf("error checking for fixed artifacts: %w", err)
	}

	// Check if artifacts are failing due to missing dependencies.
	isDependencyError, err := CheckDependencyErrors(failedArtifacts)
	if err != nil {
		return fmt.Errorf("error checking dependency errors: %w", err)
	}

	// Print each artifact's Marshalled JSON.
	for _, artifact := range failedArtifacts {
		jsonBytes, err := json.Marshal(ArtifactOutput{
			Name:                  artifact.Name(),
			Version:               artifact.Version(),
			Namespace:             artifact.Ingredients[0].Namespace,
			IsBuildtimeDependency: artifact.IsBuildtimeDependency,
			IsRuntimeDependency:   artifact.IsRuntimeDependency,
			LogURL:                artifact.LogURL,
			SourceURI:             artifact.Ingredients[0].IngredientSource.Url.String(),
			WasFixed:              wasFixed[artifact.ArtifactID],
			IsDependencyError:     isDependencyError[artifact.ArtifactID],
		})
		if err != nil {
			return fmt.Errorf("error marshaling results: %w", err)
		}
		runner.output.Print(string(jsonBytes))
	}
	return nil
}

// Check whether a newer revision is available for each artifact version.
// If found, assume the issue is resolved so that a new build could be retried.
func CheckDependencyFixes(auth *authentication.Auth, failedArtifacts []*buildplan.Artifact) (map[strfmt.UUID]bool, error) {
	fixed := make(map[strfmt.UUID]bool)
	latest, err := model.FetchLatestRevisionTimeStamp(auth)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch latest timestamp: %w", err)
	}
	for _, artifact := range failedArtifacts {
		// TODO: Query multiple artifacts at once to reduce API calls, improving performance.
		latestRevision, err := model.GetIngredientByNameAndVersion(
			artifact.Ingredients[0].Namespace, artifact.Name(), artifact.Version(), &latest, auth)
		if err != nil {
			return nil, fmt.Errorf("error searching ingredient: %w", err)
		}
		fixed[artifact.ArtifactID] = artifact.Revision() < int(*latestRevision.Revision)
	}
	return fixed, nil
}

// Perform asynchronous checks for dependency errors to improve efficiency in large projects with multiple failures.
func CheckDependencyErrors(failedArtifacts []*buildplan.Artifact) (map[strfmt.UUID]bool, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	client := &http.Client{Timeout: 30 * time.Second}

	dependencyErrors := make(map[strfmt.UUID]bool)
	for i := range failedArtifacts {
		wg.Add(1)
		go func(artifact *buildplan.Artifact) {
			defer wg.Done()
			depError, err := CheckWasDependencyError(client, artifact.LogURL)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("error checking dependency error for %s: %w", artifact.Name(), err))
				mu.Unlock()
				return
			}
			dependencyErrors[artifact.ArtifactID] = depError
		}(failedArtifacts[i])
	}

	wg.Wait()

	if len(errors) > 0 {
		return nil, fmt.Errorf("multiple errors occurred: %v", errors)
	}

	return dependencyErrors, nil
}

// For now, dependency errors are detected by downloading the logs and scanning for ModuleNotFoundError.
// Not perfect, but sufficient for Python and prevents sending large logs to the LLM prematurely.
// TODO: Implement a more robust solution that considers other languages. Ideally, this returns an error
// type, were 'dependency' is just one of them; this will give the caller an overview of failure categories.
func CheckWasDependencyError(client *http.Client, logURL string) (bool, error) {
	resp, err := client.Get(logURL)
	if err != nil {
		return false, fmt.Errorf("error checking dependency error for %s: %w", logURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, logURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("error reading response body for %s: %w", logURL, err)
	}

	return strings.Contains(string(body), "ModuleNotFoundError"), nil
}
