package testhelper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/autarch/testify/require"
)

func dataPath(t *testing.T) string {
	root, err := environment.GetRootPath()
	require.NoError(t, err)
	return filepath.Join(root, "pkg", "platform", "runtime2", "testhelper", "data")
}

func LoadRecipe(t *testing.T, name string) *inventory_models.Recipe {
	d, err := ioutil.ReadFile(filepath.Join(dataPath(t), "recipes", fmt.Sprintf("%s.json", name)))
	require.NoError(t, err)

	var recipe inventory_models.Recipe
	err = json.Unmarshal(d, &recipe)
	require.NoError(t, err)

	return &recipe
}

func LoadBuildResponse(t *testing.T, name string) *headchef_models.BuildStatusResponse {
	d, err := ioutil.ReadFile(filepath.Join(dataPath(t), "builds", fmt.Sprintf("%s.json", name)))
	require.NoError(t, err)

	var status headchef_models.BuildStatusResponse
	err = json.Unmarshal(d, &status)
	require.NoError(t, err)

	return &status
}
