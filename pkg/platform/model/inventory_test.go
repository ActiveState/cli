package model_test

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
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


func (suite *InventoryTestSuite) TestFetchPlatformByUID() {
	suite.invMock.MockPlatforms()

	uid := strfmt.UUID("00010001-0001-0001-0001-000100010001")
	platform, fail := model.FetchPlatformByUID(uid)
	suite.Require().NoError(fail.ToError())
	suite.Equal(uid, *platform.PlatformID, "Returns platform")
}

func TestInventorySuite(t *testing.T) {
	suite.Run(t, new(InventoryTestSuite))
}
