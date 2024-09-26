package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ApiIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ApiIntegrationTestSuite) TestRequestHeaders() {
	suite.OnlyRunForTags(tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/Empty", "."),
		e2e.OptAppendEnv(constants.DebugServiceRequestsEnvVarName+"=true", "VERBOSE=true"),
	)
	// e.g. User-Agent: state/0.38.0-SHA0deadbeef0; release (Windows; 10.0.22621; x86_64)
	cp.ExpectRe(`User-Agent: state/(\d+\.?)+-SHA[[:xdigit:]]+; \S+ \([^;]+; [^;]+; [^)]+\)`)
	cp.ExpectRe(`X-Requestor: [[:xdigit:]-]+`) // UUID
	cp.ExpectExitCode(0)
}

func TestApiIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ApiIntegrationTestSuite))
}
