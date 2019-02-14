package selfupdate

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/updatemocks"
	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	assert := assert.New(t)

	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()

	updatemocks.MockUpdater(t, os.Args[0], "1.2.3-123")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"self-update"})

	Command.Execute()

	assert.Equal(true, true, "Execute didn't panic")
	assert.NoError(failures.Handled(), "No failure occurred")
}
