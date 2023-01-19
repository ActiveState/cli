package mock

import (
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
)

type Mock struct {
	httpmock *httpmock.HTTPMock
}

var mock *httpmock.HTTPMock

func Init() *Mock {
	return &Mock{
		httpmock.Activate(api.GetServiceURL(api.ServiceInventory).String()),
	}
}

func (m *Mock) Close() {
	httpmock.DeActivate()
}

func (m *Mock) MockPlatforms() {
	m.httpmock.RegisterWithResponse("GET", "/v1/platforms", 200, "platforms")
}

func (m *Mock) MockOrderRecipes() {
	m.httpmock.RegisterWithResponse("POST", "/v1/recipes", 200, "recipes")
}

func (m *Mock) MockIngredientsByName() {
	m.httpmock.RegisterWithResponse("GET", "/v1/namespaces/ingredients", 200, "ingredients_by_name")
}

func (m *Mock) MockSolutions() {
	m.httpmock.RegisterWithResponse("POST", "/v1/solutions", 201, "solutions")
}
