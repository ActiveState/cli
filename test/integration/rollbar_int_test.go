package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
)

func TestRollbarReportConfig(t *testing.T) {
	ts := e2e.New(t, true)
	defer ts.Close()

	cp := ts.Spawn("--version")
	cp.Expect("Version")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config", "set", constants.ReportErrorsConfig, "false")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)
}
