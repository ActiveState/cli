package conpty

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ConPtyTestSuite struct {
	suite.Suite
}

func (suite *ConPtyTestSuite) SetupTest() {
}

func (suite *ConPtyTestSuite) BeforeTest(suiteName, testName string) {
}

func (suite *ConPtyTestSuite) AfterTest(suiteName, testName string) {
}

func (suite *ConPtyTestSuite) TestName() {

}

func TestConPtyTestSuite(t *testing.T) {
	suite.Run(t, new(ConPtyTestSuite))
}
