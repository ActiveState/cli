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
	m.httpmock.Register("GET", "/platforms")
}

func (m *Mock) MockOrderRecipes() {
	m.httpmock.Register("POST", "/orders/00010001-0001-0001-0001-000100010001/recipes")
}
