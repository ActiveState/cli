package termsize_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TermsizeTestSuite struct {
	suite.Suite
}

func (suite *TermsizeTestSuite) TestTermsize() {
	// Cannot test this because it is heavily reliant on the invoker of the test.
}

func TestTermsizeTestSuite(t *testing.T) {
	suite.Run(t, new(TermsizeTestSuite))
}
