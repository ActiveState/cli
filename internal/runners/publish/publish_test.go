package publish

import (
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
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
			Conditions: []*inventory_models.DependencyCondition{
				{Condition: inventory_models.Condition{
					Feature:   ptr.To("windows"),
					Namespace: ptr.To("kernel"),
				}},
				{Condition: inventory_models.Condition{
					Feature:   ptr.To("python"),
					Namespace: ptr.To("language"),
					Requirements: inventory_models.Requirements{
						{Comparator: ptr.To(inventory_models.RequirementComparatorLt), Version: ptr.To("3.11")},
					},
				}},
			},
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
		want := []request.Dependency{
			{Name: "windows", Namespace: "kernel"},
			{Name: "python", Namespace: "language", VersionRequirements: "<3.11"},
		}
		if c := r.Dependencies[0].Conditions; !reflect.DeepEqual(c, want) {
			t.Errorf("conditions not inherited: got %+v, want %+v", c, want)
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

// TestApplyPyprojectDependencies covers --edit with a pyproject.toml whose dependencies are all skipped: the
// dependencies inherited from the previous revision must be kept rather than replaced by an empty list.
func TestApplyPyprojectDependencies(t *testing.T) {
	inherited := []request.PublishVariableDep{{Dependency: request.Dependency{Name: "six", Namespace: "language/python"}}}
	r := &Runner{out: outputhelper.NewCatcher()}

	for _, requires := range [][]string{nil, {`colorama; platform_machine == "x86_64"`}} {
		reqVars := &request.PublishVariables{Dependencies: inherited}
		if err := r.applyPyprojectDependencies(reqVars, requires); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(reqVars.Dependencies, inherited) {
			t.Errorf("requires %q: inherited dependencies were replaced: %+v", requires, reqVars.Dependencies)
		}
	}
}

func TestCheckEditKeepsIdentity(t *testing.T) {
	withAuthor := &request.PublishVariables{Authors: []request.PublishVariableAuthor{{Name: "a", Email: "a@example.com"}}}
	withDescription := &request.PublishVariables{Description: "changed"}
	plain := &request.PublishVariables{}

	tests := []struct {
		name             string
		vars             *request.PublishVariables
		edit, isRevision bool
		wantErr          bool
	}{
		{"author on revision", withAuthor, true, true, true},
		{"description on revision is left to keepDescription", withDescription, true, true, false},
		{"no identity change on revision", plain, true, true, false},
		{"author on new version via edit", withAuthor, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkEditKeepsIdentity(tt.vars, tt.edit, tt.isRevision)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFillVersionRequirements(t *testing.T) {
	r := &request.PublishVariables{Dependencies: []request.PublishVariableDep{
		{
			Dependency: request.Dependency{Name: "requests", Namespace: "language/python"},
			Conditions: []request.Dependency{
				{Name: "Windows", Namespace: "kernel"},
				{Name: "python", Namespace: "language", VersionRequirements: "<3.11"},
			},
		},
		{Dependency: request.Dependency{Name: "six", Namespace: "language/python", VersionRequirements: ">=1.16"}},
	}}
	fillVersionRequirements(r)

	if got := r.Dependencies[0].VersionRequirements; got != ">=0" {
		t.Errorf("dependency without a version = %q, want >=0", got)
	}
	if got := r.Dependencies[0].Conditions[0].VersionRequirements; got != ">=0" {
		t.Errorf("condition without a version = %q, want >=0", got)
	}
	if got := r.Dependencies[0].Conditions[1].VersionRequirements; got != "<3.11" {
		t.Errorf("condition version changed to %q", got)
	}
	if got := r.Dependencies[1].VersionRequirements; got != ">=1.16" {
		t.Errorf("dependency version changed to %q", got)
	}
}

func TestRerunCommand(t *testing.T) {
	tests := []struct {
		params *Params
		want   string
	}{
		{&Params{Build: "./src", Filepath: "/tmp/built.tar.gz"}, "state publish --build ./src --meta m.yaml"},
		{&Params{Edit: true, Build: "my src"}, `state publish --edit --build "my src" --meta m.yaml`},
		{&Params{Filepath: "pkg.tar.gz", Name: "ignored"}, "state publish pkg.tar.gz --meta m.yaml"},
	}
	for _, tt := range tests {
		if got := rerunCommand(tt.params, "m.yaml"); got != tt.want {
			t.Errorf("rerunCommand(%+v) = %q, want %q", tt.params, got, tt.want)
		}
	}
}
