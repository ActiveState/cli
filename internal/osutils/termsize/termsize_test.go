package termsize_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/osutils/termsize"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
)

type TermsizeTestSuite struct {
	suite.Suite
}

func (suite *TermsizeTestSuite) TestTermsize() {
	suite.NotPanics(func() { termsize.GetTerminalColumns() }, "No panic should occur")
}

func TestTermsizeTestSuite(t *testing.T) {
	suite.Run(t, new(TermsizeTestSuite))
}
