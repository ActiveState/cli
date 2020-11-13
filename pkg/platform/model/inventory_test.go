package model_test

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
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

func ingredientFromName(name string) *model.IngredientAndVersion {
	n := name
	return &model.IngredientAndVersion{
		V1SearchIngredientsResponseIngredientsItems: &inventory_models.V1SearchIngredientsResponseIngredientsItems{
			Ingredient: &inventory_models.V1SearchIngredientsResponseIngredientsItemsIngredient{
				V1SearchIngredientsResponseIngredientsItemsIngredientAllOf1: inventory_models.V1SearchIngredientsResponseIngredientsItemsIngredientAllOf1{
					V1SearchIngredientsResponseIngredientsItemsIngredientAllOf1AllOf0: inventory_models.V1SearchIngredientsResponseIngredientsItemsIngredientAllOf1AllOf0{
						Name: &n,
					},
				},
			},
		},
		Version: "1.0",
	}
}

func (suite *InventoryTestSuite) TestfilterCandidates() {
	datetime := ingredientFromName("datetime")
	DateTime := ingredientFromName("DateTime")
	sample := []*model.IngredientAndVersion{datetime, DateTime}

	cases := []struct {
		name      string
		arg       string
		expected  *model.IngredientAndVersion
		wantError bool
	}{
		{"lower-case", "datetime", datetime, false},
		{"upper-case", "DateTime", DateTime, false},
		{"error", "DateTIME", nil, true},
	}

	for _, c := range cases {
		suite.Run(c.name, func() {
			res, err := model.FilterForBestIngredientMatch(sample, c.arg)
			suite.Assert().Equal(res, c.expected)
			if (err != nil) != c.wantError {
				suite.T().Fatalf("filterForBestIngredientMatch returned unexpected error value: %v", err)
			}
		})
	}
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
