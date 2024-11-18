package response

import "github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"

// RemediableSolveError represents a solver error that can be remediated.
type RemediableError struct {
	Remediations []*SolverErrorRemediation `json:"possibleRemediations"`
}

type GenericSolveError struct {
	IsTransient      bool                          `json:"isTransient"`
	ValidationErrors []*SolverErrorValidationError `json:"validationErrors"`
}

// SolverErrorValidationError represents a validation error that occurred during planning.
type SolverErrorValidationError struct {
	JSONPath string `json:"jsonPath"`
	Error    string `json:"error"`
}

// SolverErrorRemediation contains the recommeneded remediation for remediable error.
type SolverErrorRemediation struct {
	Description       string `json:"description"`
	SuggestedPriority string `json:"suggestedPriority"`
}

type RemediableSolveError struct {
	ErrorType             string                       `json:"errorType"`
	Incompatibilities     []*SolveErrorIncompatibility `json:"incompatibilities"`
	Requirements          []*types.Requirement         `json:"requirements"`
	SuggestedRemediations []*SolverErrorRemediation    `json:"suggestedRemediations"`
}

// SolverErrorIncompatibility represents a solver incompatibility error.
type SolveErrorIncompatibility struct {
	Type string `json:"type"`
	*SolveErrorPackageIncompatibility
	*SolveErrorPlatformIncompatibility
}

// SolveErrorPackageIncompatibility represents a package incompatibility error.
type SolveErrorPackageIncompatibility struct {
	Type      string `json:"type"`
	Feature   string `json:"feature"`
	Namespace string `json:"namespace"`
}

// SolveErrorPlatformIncompatibility represents a platform incompatibility error.
type SolveErrorPlatformIncompatibility struct {
	Type           string `json:"type"`
	PlatformID     string `json:"platformID"`
	PlatformKernel string `json:"platformKernel"`
}

const (
	SolveErrorIncompatibilityTypeDependency  = "DEPENDENCY"
	SolveErrorIncompatibilityTypePlatform    = "PLATFORM"
	SolveErrorIncompatibilityTypeRequirement = "REQUIREMENT"
)
