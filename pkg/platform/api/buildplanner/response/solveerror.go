package response

import (
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

// SolverErrorValidationError represents a validation error that occurred during planning.
type SolverErrorValidationError struct {
	JSONPath string `json:"jsonPath"`
	Error    string `json:"error"`
}

// RemediableSolveError represents a solver error that can be remediated.
type RemediableSolveError struct {
	ErrorType         string                       `json:"errorType"`
	Remediations      []*SolverErrorRemediation    `json:"suggestedRemediations"`
	Requirements      []*types.Requirement         `json:"requirements"`
	Incompatibilities []*SolveErrorIncompatibility `json:"incompatibilities"`
}

// SolverErrorRemediation contains the recommeneded remediation for remediable error.
type SolverErrorRemediation struct {
	RemediationType string `json:"solveErrorRemediationType"`
	Command         string `json:"command"`
}

// SolverErrorIncompatibility represents a solver incompatibility error.
type SolveErrorIncompatibility struct {
	Type string `json:"type"`
	*SolveErrorPackageIncompatibility
	*SolveErrorPlatformIncompatibility
}

// SolveErrorPackageIncompatibility represents a package incompatibility error.
type SolveErrorPackageIncompatibility struct {
	Feature   string `json:"feature"`
	Namespace string `json:"namespace"`
}

// SolveErrorPlatformIncompatibility represents a platform incompatibility error.
type SolveErrorPlatformIncompatibility struct {
	PlatformID     string `json:"platformID"`
	PlatformKernel string `json:"platformKernel"`
}
