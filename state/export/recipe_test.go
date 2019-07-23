package export

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func setupRecipeCommand(t *testing.T, args ...string) func() {
	root, err := environment.GetRootPath()
	require.NoError(t, err, "should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	cc := Command.GetCobraCmd()
	cc.SetArgs(append([]string{"recipe"}, args...))

	apim := apiMock.Init()
	authm := authMock.Init()
	invm := invMock.Init()

	cleanup := func() {
		invm.Close()
		authm.Close()
		apim.Close()
	}

	authm.MockLoggedin()
	apim.MockGetProject()
	apim.MockVcsGetCheckpoint()
	invm.MockPlatforms()
	invm.MockOrderRecipes()

	return cleanup
}

func TestExportRecipe(t *testing.T) {
	cleanup := setupRecipeCommand(t, "test")
	defer cleanup()

	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	ex := exiter.New()
	Command.Exiter = ex.Exit
	exitCode := ex.WaitForExit(func() {
		Command.Execute()
	})

	assert.Equal(t, 0, exitCode, "exited with code 0")
}
