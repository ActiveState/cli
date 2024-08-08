package response

import (
	"errors"

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

const buildPlannedType = "BuildPlanned"
const buildPlanningType = "BuildPlanning"

type ImpactReportBuildResult struct {
	Type string `json:"__typename"`
}

type ImpactReportBuildResultError struct {
	BuildBefore *ImpactReportBuildResult `json:"buildBefore"`
	BuildAfter  *ImpactReportBuildResult `json:"buildAfter"`
}

type ImpactReportResult struct {
	Type        string                   `json:"__typename"`
	Ingredients []ImpactReportIngredient `json:"ingredients"`
	*Error
	*ImpactReportBuildResultError
}

type ImpactReportResponse struct {
	*ImpactReportResult `json:"impactReport"`
}

type ImpactReportError struct {
	Type    string
	Message string
}

func (e ImpactReportError) Error() string { return e.Message }

func IsImpactReportBuildPlanningError(err error) bool {
	var impactReportErr *ImpactReportError
	return errors.As(err, &impactReportErr) && impactReportErr.Type == buildPlanningType
}

func ProcessImpactReportError(err *ImpactReportResult, fallbackMessage string) error {
	if err.Error == nil {
		return errs.New(fallbackMessage)
	}

	errType := err.Type
	message := err.Message
	buildResultError := err.ImpactReportBuildResultError
	if beforeType := buildResultError.BuildBefore.Type; beforeType != buildPlannedType {
		errType = beforeType
		message += "\nbuildBefore status: " + beforeType
	} else if afterType := buildResultError.BuildAfter.Type; afterType != buildPlannedType {
		errType = afterType
		message += "\nbuildAfter status: " + afterType
	}

	return &ImpactReportError{errType, message}
}
