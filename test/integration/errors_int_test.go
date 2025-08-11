package integration

import (
	"testing"

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

	cp := ts.Spawn("__test", "multierror-input")
	cp.ExpectRe(`\s+x error1.\s+\s+x error2.\s+x error3.\s+x error4.\s+`)
	cp.ExpectExitCode(1)
}

func (suite *ErrorsIntegrationTestSuite) TestMultiErrorWithoutInput() {
	suite.OnlyRunForTags(tagsuite.Errors, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.IgnoreLogErrors()

	cp := ts.Spawn("__test", "multierror-noinput")
	cp.ExpectRe(`\s+x error1.\s+\s+x error2.\s+x error3.\s+x error4.\s+`)
	cp.ExpectExitCode(1)
}

func TestErrorsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorsIntegrationTestSuite))
}
