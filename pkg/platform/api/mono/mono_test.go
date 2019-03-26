package mono_test

import (
	"net/http"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/authentication"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/status"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")

	client := mono.New()
	_, err := client.Authentication.PostLogin(authentication.NewPostLoginParams())
	assert.NoError(t, err)
}

func TestNewWithAuth(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/info")

	authInfo := httptransport.BearerToken("")
	client := mono.NewWithAuth(&authInfo)
	_, err := client.Status.GetInfo(status.NewGetInfoParams(), authInfo)
	assert.NoError(t, err)
}

func TestUserAgent(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	var userAgent string
	httpmock.RegisterWithResponder("POST", "/login", func(req *http.Request) (int, string) {
		userAgent = req.Header.Get("User-Agent")
		return 200, "login"
	})

	client := mono.New()
	_, err := client.Authentication.PostLogin(authentication.NewPostLoginParams())
	require.NoError(t, err)
	assert.Contains(t, userAgent, constants.UserAgent)
}

func TestPersist(t *testing.T) {
	client := mono.Get()
	client2 := mono.Get()
	assert.True(t, client == client2, "Should return same pointer")
}
