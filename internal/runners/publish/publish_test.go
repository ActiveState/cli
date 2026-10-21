package publish

import (
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
)

func TestPrepareEditRequestDependencies(t *testing.T) {
	ingredient := &ParentIngredient{
		Version: "1.0.0",
		Dependencies: []inventory_models.Dependency{{
			Feature:   ptr.To("python"),
			Namespace: ptr.To("language"),
			Type:      inventory_models.DependencyTypeRuntime,
			Conditions: []*inventory_models.DependencyCondition{{Condition: inventory_models.Condition{
				Feature:   ptr.To("windows"),
				Namespace: ptr.To("kernel"),
			}}},
		}},
	}

	t.Run("inherits when none supplied", func(t *testing.T) {
		r := &request.PublishVariables{Version: "9.9.9"}
		if err := prepareEditRequest(ingredient, r, true, nil); err != nil {
			t.Fatal(err)
		}
		if len(r.Dependencies) != 1 || r.Dependencies[0].Name != "python" || r.Dependencies[0].Type != request.DependencyTypeRuntime {
			t.Errorf("dependencies not inherited: %+v", r.Dependencies)
		}
		if c := r.Dependencies[0].Conditions; len(c) != 1 || c[0].Name != "windows" || c[0].Namespace != "kernel" {
			t.Errorf("conditions not inherited: %+v", c)
		}
		if r.Version != "1.0.0" {
			t.Errorf("version = %q, an edit must keep the ingredient's version", r.Version)
		}
	})

	t.Run("meta file dependencies are kept", func(t *testing.T) {
		metaDep := request.PublishVariableDep{Dependency: request.Dependency{Name: "six", Namespace: "language/python"}}
		r := &request.PublishVariables{Dependencies: []request.PublishVariableDep{metaDep}}
		if err := prepareEditRequest(ingredient, r, true, nil); err != nil {
			t.Fatal(err)
		}
		if len(r.Dependencies) != 1 || r.Dependencies[0].Dependency != metaDep.Dependency {
			t.Errorf("meta file dependencies were overwritten: %+v", r.Dependencies)
		}
	})
}
