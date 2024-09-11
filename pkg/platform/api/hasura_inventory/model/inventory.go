package model

import (
	"github.com/go-openapi/strfmt"
)

type LastIngredientRevisionTime struct {
	RevisionTime strfmt.DateTime `json:"revision_time"`
}

type LatestRevisionResponse struct {
	RevisionTimes []LastIngredientRevisionTime `json:"last_ingredient_revision_time"`
}

type Namespace struct {
	Namespace string `json:"namespace"`
}

type IngredientVersion struct {
	Version             string      `json:"version"`
	IngredientVersionID strfmt.UUID `json:"ingredient_version_id"`
	LicenseExpression   string      `json:"license_expression"`
}

type SearchIngredient struct {
	Name           string              `json:"name"`
	NormalizedName string              `json:"normalized_name"`
	Namespace      Namespace           `json:"namespace"`
	IngredientID   strfmt.UUID         `json:"ingredient_id"`
	Description    string              `json:"description"`
	Website        *string             `json:"website"`
	Versions       []IngredientVersion `json:"versions"`
}

type SearchIngredientsResponse struct {
	SearchIngredients []SearchIngredient `json:"search_ingredients"`
}
