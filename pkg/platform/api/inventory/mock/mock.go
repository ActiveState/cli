package mock

import (
	"runtime"

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
	if runtime.GOOS == "windows" {
		m.httpmock.RegisterWithResponse("GET", "/platforms", 200, "platforms-win")
	} else {
		// For development we'll sometimes spoof linux on mac, so be agnostic
		m.httpmock.RegisterWithResponse("GET", "/platforms", 200, "platforms")
	}
}

func (m *Mock) MockOrderRecipes() {
	m.httpmock.Register("POST", "/orders/00010001-0001-0001-0001-000100010001/recipes")
	m.httpmock.Register("POST", "/orders/00020002-0002-0002-0002-000200020002/recipes")
}

func (m *Mock) MockIngredientsByName() {
	m.httpmock.RegisterWithResponse("GET", "/ingredients?package_name=artifact", 200, "ingredients_by_name")
}
