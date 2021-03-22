package branch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/stretchr/testify/assert"
)

func TestBranchListing_Simple(t *testing.T) {
	branches := getBranches(t, "simple")
	out := NewBranchOutput(branches, "")
	actual := out.MarshalOutput(output.PlainFormatName)
	expected := " [NOTICE]main[/RESET]\n"
	assert.Equal(t, expected, actual)
}

func TestBranchListing_Complex(t *testing.T) {
	branches := getBranches(t, "complex")
	out := NewBranchOutput(branches, "")
	actual := out.MarshalOutput(output.PlainFormatName)
	expected := ` [NOTICE]main[/RESET]
  ├─ [NOTICE]subBranch1[/RESET]
  │  ├─ [NOTICE]childOfSubBranch1[/RESET]
  │  │  └─ [NOTICE]3rdLevelChild[/RESET]
  │  └─ [NOTICE]secondChildOfSubBranch1[/RESET]
  ├─ [NOTICE]subBranch2[/RESET]
  └─ [NOTICE]subBranch3[/RESET]
`
	assert.Equal(t, expected, actual)
}

func TestBranchListing_MultipleRoots(t *testing.T) {
	branches := getBranches(t, "multipleRoots")
	out := NewBranchOutput(branches, "root1Child2")
	actual := out.MarshalOutput(output.PlainFormatName)
	expected := ` [NOTICE]root1[/RESET]
 [NOTICE]root2[/RESET]
  ├─ [NOTICE]root1Child1[/RESET]
  ├─ [ACTIONABLE]root1Child2[/RESET] [DISABLED](Current)[/RESET]
  │  └─ [NOTICE]childOfRoot1Child2[/RESET]
  └─ [NOTICE]root2Child3[/RESET]
 [NOTICE]root3[/RESET]
`
	assert.Equal(t, expected, actual)
}

func getBranches(t *testing.T, testName string) mono_models.Branches {
	root, err := environment.GetRootPath()
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadFile(filepath.Join(root, "internal", "runners", "branch", "testdata", fmt.Sprintf("%s.json", testName)))
	if err != nil {
		t.Fatal(err)
	}

	var branches mono_models.Branches
	err = json.Unmarshal(data, &branches)
	if err != nil {
		t.Fatal(err)
	}

	return branches
}
