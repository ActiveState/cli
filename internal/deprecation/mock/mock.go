package mock

import (
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
)

type Mock struct {
	httpPrefix string
	httpSuffix string
	httpmock   *httpmock.HTTPMock
}

func Init() *Mock {
	u, err := url.Parse(constants.DeprecationInfoURL)
	if err != nil {
		log.Panicf("%v", err)
	}

	mock := &Mock{}

	mock.httpSuffix = filepath.Base(u.Path)
	mock.httpPrefix = strings.TrimSuffix(u.String(), mock.httpSuffix)
	mock.httpmock = httpmock.Activate(mock.httpPrefix)

	return mock
}

func (m *Mock) Close() {
	httpmock.DeActivate()
}

func (m *Mock) MockEmpty() {
	m.httpmock.RegisterWithResponseBody("GET", m.httpSuffix, 200, `[]`)
}

func (m *Mock) MockExpired() {
	m.httpmock.RegisterWithResponseBody("GET", m.httpSuffix, 200, `
	[
		{"version": "111.0.0", "date": "1970-01-01T00:00:00Z", "reason": "Some reason"},
		{"version": "222.0.0", "date": "1970-01-01T00:00:00Z", "reason": "Some reason"},
		{"version": "999.0.0", "date": "1970-01-01T00:00:00Z", "reason": "Some reason"},
		{"version": "333.0.0", "date": "1970-01-01T00:00:00Z", "reason": "Some reason"}
	]`)
}

func (m *Mock) MockExpiredTimed(duration time.Duration) {
	m.httpmock.RegisterWithResponderBody("GET", m.httpSuffix, func(req *http.Request) (int, string) {
		time.Sleep(duration)
		return 200, `[{ "version": "111.0.0", "date": "1970-01-01T00:00:00Z", "reason": "Some reason" }]`
	})
}

func (m *Mock) MockDeprecated() {
	m.httpmock.RegisterWithResponseBody("GET", m.httpSuffix, 200, `[{ "version": "999.0.0", "date": "2222-01-01T00:00:00Z", "reason": "Some reason" }]`)
}
