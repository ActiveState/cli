package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
)

func setupRecipeCommand(t *testing.T, args ...string) func() {
	root, err := environment.GetRootPath()
	require.NoError(t, err, "should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	cc := Command.GetCobraCmd()
	cc.SetArgs(append([]string{"recipe"}, args...))

	im := invMock.Init()
	am := authMock.Init()
	cleanup := func() {
		im.Close()
		am.Close()
	}

	im.MockPlatforms()
	im.MockOrderRecipes()

	am.MockLoggedin()

	return cleanup
}

func TestExportRecipe(t *testing.T) {
	cleanup := setupRecipeCommand(t, "test")
	defer cleanup()

	ex := exiter.New()
	Command.Exiter = ex.Exit
	exitCode := ex.WaitForExit(func() {
		Command.Execute()
	})

	assert.Equal(t, 0, exitCode, "exited with code 0")
}
