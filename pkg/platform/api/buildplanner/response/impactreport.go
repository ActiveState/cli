package response

import (
	"github.com/ActiveState/cli/internal/errs"
)

type ImpactReportIngredientState struct {
	IngredientID  string `json:"ingredientID"`
	Version       string `json:"version"`
	IsRequirement bool   `json:"isRequirement"`
}

type ImpactReportIngredient struct {
	Namespace string                       `json:"namespace"`
	Name      string                       `json:"name"`
	Before    *ImpactReportIngredientState `json:"before"`
	After     *ImpactReportIngredientState `json:"after"`
}

type ImpactReportResult struct {
	Type        string                   `json:"__typename"`
	Ingredients []ImpactReportIngredient `json:"ingredients"`
	*Error
}

type ImpactReportResponse struct {
	*ImpactReportResult `json:"impactReport"`
}

type ImpactReportError struct {
	Type    string
	Message string
}

func (e ImpactReportError) Error() string { return e.Message }

func ProcessImpactReportError(err *ImpactReportResult, fallbackMessage string) error {
	if err.Error == nil {
		return errs.New(fallbackMessage)
	}

	return &ImpactReportError{err.Type, err.Message}
}
