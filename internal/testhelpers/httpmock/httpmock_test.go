package httpmock

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

var prefix = "http://test.tld/"

func TestMock(t *testing.T) {
	Activate(prefix)
	defer DeActivate()

	Register("GET", "test")
	resp, err := http.Get(prefix + "test")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err, "Can call configured http mock")
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, `{ "ok": true }`, string(body), "Returns the expected body")

	RegisterWithCode("GET", "test", 501)
	resp, err = http.Get(prefix + "test")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err, "Can call configured http mock")
	assert.Equal(t, 501, resp.StatusCode)
	assert.Equal(t, `{ "501": true }`, string(body), "Returns the expected body")

	RegisterWithResponse("GET", "test", 501, "custom")
	resp, err = http.Get(prefix + "test")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err, "Can call configured http mock")
	assert.Equal(t, 501, resp.StatusCode)
	assert.Equal(t, `{ "custom": true }`, string(body), "Returns the expected body")

	RegisterWithResponseBody("GET", "test", 501, "body")
	resp, err = http.Get(prefix + "test")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err, "Can call configured http mock")
	assert.Equal(t, 501, resp.StatusCode)
	assert.Equal(t, `body`, string(body), "Returns the expected body")

	RegisterWithResponseBytes("GET", "test", 501, []byte("body"))
	resp, err = http.Get(prefix + "test")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err, "Can call configured http mock")
	assert.Equal(t, 501, resp.StatusCode)
	assert.Equal(t, `body`, string(body), "Returns the expected body")
}
