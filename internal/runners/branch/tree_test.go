package branch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/stretchr/testify/assert"
)

func TestBranchListing_Simple(t *testing.T) {
	branches := getBranches(t, "simple")
	tree := NewBranchTree()
	tree.BuildFromBranches(branches)
	actual := tree.String()
	expected := "main\n"
	assert.Equal(t, expected, actual)
}

func TestBranchListing_Complex(t *testing.T) {
	branches := getBranches(t, "complex")
	tree := NewBranchTree()
	tree.BuildFromBranches(branches)
	actual := tree.String()
	expected := `main
 ├─ subBranch1
 │  ├─ childOfSubBranch1
 │  │  └─ 3rdLevelChild
 │  └─ secondChildOfSubBranch1
 ├─ subBranch2
 └─ subBranch3
`
	assert.Equal(t, expected, actual)
}

func TestBranchListing_MultipleRoots(t *testing.T) {
	branches := getBranches(t, "multipleRoots")
	tree := NewBranchTree()
	tree.BuildFromBranches(branches)
	actual := tree.String()
	expected := `root1
root2
 ├─ root1Child1
 ├─ root1Child2
 │  └─ childOfRoot1Child2
 └─ root2Child3
root3
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
