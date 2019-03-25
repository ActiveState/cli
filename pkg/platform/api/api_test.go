package api_test

import (
	"net/http"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/authentication"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/status"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")

	client := api.New()
	_, err := client.Authentication.PostLogin(authentication.NewPostLoginParams())
	assert.NoError(t, err)
}

func TestNewWithAuth(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/info")

	authInfo := httptransport.BearerToken("")
	client := api.NewWithAuth(&authInfo)
	_, err := client.Status.GetInfo(status.NewGetInfoParams(), authInfo)
	assert.NoError(t, err)
}

func TestUserAgent(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())
	defer httpmock.DeActivate()

	var userAgent string
	httpmock.RegisterWithResponder("POST", "/login", func(req *http.Request) (int, string) {
		userAgent = req.Header.Get("User-Agent")
		return 200, "login"
	})

	client := api.New()
	_, err := client.Authentication.PostLogin(authentication.NewPostLoginParams())
	require.NoError(t, err)
	assert.Contains(t, userAgent, constants.UserAgent)
}

func TestPersist(t *testing.T) {
	client := api.Get()
	client2 := api.Get()
	assert.True(t, client == client2, "Should return same pointer")
}

func TestErrorCode_WithoutPayload(t *testing.T) {
	assert.Equal(t, 100, api.ErrorCode(&struct{ Code int }{
		Code: 100,
	}))
}

func TestErrorCode_WithoutPayload_NoCodeValue(t *testing.T) {
	assert.Equal(t, -1, api.ErrorCode(&struct{ OtherCode int }{
		OtherCode: 100,
	}))
}

func TestErrorCode_WithPayload(t *testing.T) {
	providedCode := 200
	codeValue := struct{ Code *int }{Code: &providedCode}
	payload := struct{ Payload struct{ Code *int } }{
		Payload: codeValue,
	}

	assert.Equal(t, 200, api.ErrorCode(&payload))
}

func TestErrorCode_WithPayload_CodeNotPointer(t *testing.T) {
	providedCode := 300
	codeValue := struct{ Code int }{Code: providedCode}
	payload := struct{ Payload struct{ Code int } }{
		Payload: codeValue,
	}

	assert.Equal(t, 300, api.ErrorCode(&payload))
}

func TestErrorCode_WithPayload_NoCodeField(t *testing.T) {
	providedCode := 400
	codeValue := struct{ OtherCode int }{OtherCode: providedCode}
	payload := struct{ Payload struct{ OtherCode int } }{
		Payload: codeValue,
	}

	assert.Equal(t, -1, api.ErrorCode(&payload))
}
