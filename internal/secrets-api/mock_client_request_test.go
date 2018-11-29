package secretsapi_test

import (
	"net/url"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/mock"
)

func mockRecover() {
	p := recover()
	if p != nil {
		if pstr, isStr := p.(string); !isStr || !strings.Contains(pstr, "mock") {
			panic(p)
		}
	}
}

type MockClientRequest struct {
	mock.Mock
}

func (m *MockClientRequest) SetHeaderParam(name string, values ...string) error {
	defer mockRecover()
	args := m.Called(append([]interface{}{name}, values)...)
	return args.Error(0)
}

// Fill out the rest of these as needed

func (m *MockClientRequest) SetQueryParam(_ string, _ ...string) error { return nil }

func (m *MockClientRequest) SetFormParam(_ string, _ ...string) error { return nil }

func (m *MockClientRequest) SetPathParam(_ string, _ string) error { return nil }

func (m *MockClientRequest) SetFileParam(_ string, _ ...runtime.NamedReadCloser) error { return nil }

func (m *MockClientRequest) SetBodyParam(body interface{}) error { return nil }

func (m *MockClientRequest) SetTimeout(timeout time.Duration) error { return nil }

func (m *MockClientRequest) GetQueryParams() url.Values { return nil }

func (m *MockClientRequest) GetMethod() string { return "" }

func (m *MockClientRequest) GetPath() string { return "" }

func (m *MockClientRequest) GetBody() []byte { return nil }
