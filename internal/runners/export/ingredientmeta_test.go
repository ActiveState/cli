package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"gopkg.in/yaml.v3"
)

func TestWriteIngredientMeta(t *testing.T) {
	meta := &request.PublishVariables{
		Name:        "mypkg",
		Namespace:   "private/org/python",
		Version:     "1.0.0",
		Description: strings.Repeat("a long description [RED]with tags[/RESET] ", 10),
		Dependencies: []request.PublishVariableDep{{
			Dependency: request.Dependency{Name: "requests", Namespace: "language/python", VersionRequirements: ">=2"},
		}},
	}

	var out bytes.Buffer
	if err := writeIngredientMeta(&out, meta); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "#") {
		t.Errorf("meta file contains comments:\n%s", out.String())
	}

	var got request.PublishVariables
	if err := yaml.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("meta file is not valid yaml: %v\n%s", err, out.String())
	}
	if got.Description != meta.Description || len(got.Dependencies) != 1 || got.Dependencies[0].Dependency != meta.Dependencies[0].Dependency {
		t.Errorf("meta file does not round-trip:\n%s", out.String())
	}
}
