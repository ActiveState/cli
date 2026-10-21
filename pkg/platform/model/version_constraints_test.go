package model

import (
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
)

func TestInventoryRequirementsToString(t *testing.T) {
	gte := inventory_models.RequirementComparatorGte
	lt := inventory_models.RequirementComparatorLt

	tests := []struct {
		name string
		reqs inventory_models.Requirements
		want string
	}{
		{"nil", nil, ""},
		{"range", inventory_models.Requirements{
			{Comparator: &gte, Version: ptr.To("1.16")},
			{Comparator: &lt, Version: ptr.To("2")},
		}, ">=1.16,<2"},
		// Some platform ingredients carry condition requirements without a version (e.g. click's
		// language/alternative-built-language condition); they place no constraint and must not panic.
		{"missing version is skipped", inventory_models.Requirements{
			{Comparator: &gte},
			{Comparator: &lt, Version: ptr.To("3.11")},
		}, "<3.11"},
		{"missing comparator is skipped", inventory_models.Requirements{{Version: ptr.To("1.0")}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InventoryRequirementsToString(tt.reqs, true); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
