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

	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
)

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

type FailedIngredient struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	Namespace         string `json:"namespace"`
	BuildTimestamp    string `json:"build_timestamp"`
	LogURL            string `json:"logURL"`
	WasFixed          bool   `json:"wasFixed"`
	IsDependencyError bool   `json:"is_dependency_error"`
}

func (runner *ProjectErrorsRunner) Run() error {
	branch, err := model.DefaultBranchForProjectName(runner.namespace.Owner, runner.namespace.Project)
	if err != nil {
		return fmt.Errorf("error fetching default branch: %w", err)
	}

	bpm := buildplanner.NewBuildPlannerModel(runner.primer.Auth(), runner.primer.SvcModel())
	commit, err := bpm.FetchCommitNoPoll(
		strfmt.UUID(branch.CommitID.String()), runner.namespace.Owner, runner.namespace.Project, nil)
	if err != nil {
		return fmt.Errorf("error fetching commit: %w", err)
	}

	bp := commit.BuildPlan()
	failedArtifacts := bp.Artifacts(buildplan.FilterFailedArtifacts())

	// Check whether a newer revision is available for each artifact version.
	// If found, assume the issue is resolved and that a new build can be retried.
	failedIngredients := []FailedIngredient{}
	for _, artifact := range failedArtifacts {
		ingredient, err := model.GetIngredientByNameAndVersion(
			artifact.Ingredients[0].Namespace, artifact.Name(), artifact.Version(), nil, runner.primer.Auth())
		if err != nil {
			return fmt.Errorf("error searching ingredient: %w", err)
		}

		failedIngredients = append(failedIngredients, FailedIngredient{
			Name:      artifact.Name(),
			Version:   artifact.Version(),
			Namespace: artifact.Ingredients[0].Namespace,
			LogURL:    artifact.LogURL,
			WasFixed:  artifact.Revision() < int(*ingredient.Revision),
		})
	}

	// Check if ingredients are failing due to missing dependencies.
	err = CheckDependencyErrors(&failedIngredients)
	if err != nil {
		return fmt.Errorf("error checking dependency errors: %w", err)
	}

	// Marshal and output the detailed list of failing ingredients.
	jsonBytes, err := json.Marshal(failedIngredients)
	if err != nil {
		return fmt.Errorf("error marshaling results: %w", err)
	}
	runner.primer.Output().Print(string(jsonBytes))

	return nil
}

// Perform asynchronous checks for dependency errors to improve efficiency in large projects with multiple failures.
func CheckDependencyErrors(failedIngredients *[]FailedIngredient) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	client := &http.Client{Timeout: 30 * time.Second}

	for i := range *failedIngredients {
		wg.Add(1)
		go func(ingredient *FailedIngredient) {
			defer wg.Done()
			depError, err := CheckWasDependencyError(client, ingredient.LogURL)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("error checking dependency error for %s: %w", ingredient.Name, err))
				mu.Unlock()
				return
			}
			ingredient.IsDependencyError = depError
		}(&(*failedIngredients)[i])
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errors)
	}

	return nil
}

var MissingDependencyKeywords = map[string][]string{
	"language/php": {
		"require_once(): Failed opening required",
		"include(): Failed opening",
		"require(): Failed opening required",
		"include_once(): Failed opening",
		"Fatal error: Uncaught Error: Class",
		"Class not found",
	},
	"language/perl": {
		"Can't locate",
		"in @INC",
		"BEGIN failed--compilation aborted",
	},
	"language/python": {
		"ModuleNotFoundError",
		"No module named",
		"ImportError",
	},
	"language/tcl": {
		"can't find package",
		"couldn't load library",
		"package require",
	},
	"language/ruby": {
		"LoadError",
		"cannot load such file",
		"no such file to load",
		"in `require'",
	},
	"language/c-sharp": {
		"CS0246",
		"are you missing a using directive",
		"could not be found",
		"The type or namespace",
	},
	"shared": {
		"unresolved import", "no external crate", "E0432", "E0463", "can't find crate", // RUST
		"No such file or directory", "fatal error:", "undefined reference to", // C/CPP
	},
}

// Dependency errors are detected by downloading the logs and scanning for known missing dependency keywords.
// Not perfect, but sufficient for early debugging and prevents sending large logs to the LLM prematurely.
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

	// Check for missing dependency keywords across all supported languages.
	// Even if the namespace is Python, for example, a shared library might be missing a symbol.
	for _, keywords := range MissingDependencyKeywords {
		for _, kw := range keywords {
			if strings.Contains(string(body), kw) {
				return true, nil
			}
		}
	}

	return false, nil
}
