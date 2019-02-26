package model_test

import (
	"testing"

	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/stretchr/testify/suite"
)

type InventoryTestSuite struct {
	suite.Suite
	invMock *invMock.Mock
}

func (suite *InventoryTestSuite) BeforeTest(suiteName, testName string) {
	suite.invMock = invMock.Init()
}

func (suite *InventoryTestSuite) AfterTest(suiteName, testName string) {
	suite.invMock.Close()
	suite.invMock = nil
}

func (suite *InventoryTestSuite) TestGetInventory() {
	suite.invMock.MockPlatforms()

	platforms, fail := model.FetchPlatforms()
	suite.Require().NoError(fail.ToError())
	suite.NotEmpty(platforms, "Returns platforms")
}

func TestInventorySuite(t *testing.T) {
	suite.Run(t, new(InventoryTestSuite))
}
