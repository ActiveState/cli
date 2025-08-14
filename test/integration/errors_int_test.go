package integration

import (
	"testing"

	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/testhelpers/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ErrorsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ErrorsIntegrationTestSuite) TestTips() {
	suite.OnlyRunForTags(tagsuite.Errors, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.IgnoreLogErrors()

	cp := ts.Spawn("__test", "multierror")
	cp.Expect("Need More Help?")
	cp.Expect("Run →")
	cp.Expect("Ask For Help →")
	cp.ExpectExitCode(1)
}

func (suite *ErrorsIntegrationTestSuite) TestMultiErrorWithInput() {
	suite.OnlyRunForTags(tagsuite.Errors, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.IgnoreLogErrors()

	cp := ts.SpawnWithOpts(e2e.OptArgs("__test", "multierror-input"), e2e.OptTermTest(termtest.OptVerboseLogger()))
	cp.ExpectRe(`x error1.\s+\s+x error2.\s+x error3.\s+x error4.`)
	cp.ExpectExitCode(1)
}

func (suite *ErrorsIntegrationTestSuite) TestMultiErrorWithoutInput() {
	suite.OnlyRunForTags(tagsuite.Errors, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.IgnoreLogErrors()

	cp := ts.SpawnWithOpts(e2e.OptArgs("__test", "multierror-noinput"), e2e.OptTermTest(termtest.OptVerboseLogger()))
	cp.ExpectRe(`x error1.\s+\s+x error2.\s+x error3.\s+x error4.`)
	cp.ExpectExitCode(1)
}

func TestErrorsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorsIntegrationTestSuite))
}
