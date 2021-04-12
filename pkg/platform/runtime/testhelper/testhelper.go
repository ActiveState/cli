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
	"github.com/stretchr/testify/require"
)

func dataPathErr() (string, error) {
	root, err := environment.GetRootPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "pkg", "platform", "runtime", "testhelper", "data"), nil
}

func dataPath(t *testing.T) string {
	fp, err := dataPathErr()
	require.NoError(t, err)
	return fp
}

func LoadRecipe(t *testing.T, name string) *inventory_models.Recipe {
	d, err := ioutil.ReadFile(filepath.Join(dataPath(t), "recipes", fmt.Sprintf("%s.json", name)))
	require.NoError(t, err)

	var recipe inventory_models.Recipe
	err = json.Unmarshal(d, &recipe)
	require.NoError(t, err)

	return &recipe
}

func SaveRecipe(name string, m *inventory_models.Recipe) error {
	return save("recipes", name, m)
}

func save(dir, name string, m interface{}) error {
	dp, err := dataPathErr()
	if err != nil {
		return err
	}
	fn := filepath.Join(dp, dir, fmt.Sprintf("%s.json", name))

	d, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fn, d, 0666)
}

func LoadBuildResponse(t *testing.T, name string) *headchef_models.BuildStatusResponse {
	d, err := ioutil.ReadFile(filepath.Join(dataPath(t), "builds", fmt.Sprintf("%s.json", name)))
	require.NoError(t, err)

	var status headchef_models.BuildStatusResponse
	err = json.Unmarshal(d, &status)
	require.NoError(t, err)

	return &status
}

func SaveBuildResponse(name string, m *headchef_models.BuildStatusResponse) error {
	return save("builds", name, m)
}
