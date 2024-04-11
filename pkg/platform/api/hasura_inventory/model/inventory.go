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
