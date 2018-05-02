package httpmock

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var prefix = "http://test.tld/"

func TestMock(t *testing.T) {
	Activate(prefix)
	defer DeActivate()

	Register("GET", "test")
	resp, err := http.Get(prefix + "test")
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	assert.NoError(t, err, "Can call configured http mock")
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, `{ "ok": true }`, string(body), "Returns the expected body")

	RegisterWithCode("GET", "test", 501)
	resp, err = http.Get(prefix + "test")
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	assert.NoError(t, err, "Can call configured http mock")
	assert.Equal(t, 501, resp.StatusCode)
	assert.Equal(t, `{ "501": true }`, string(body), "Returns the expected body")

	RegisterWithResponse("GET", "test", 501, "custom")
	resp, err = http.Get(prefix + "test")
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	assert.NoError(t, err, "Can call configured http mock")
	assert.Equal(t, 501, resp.StatusCode)
	assert.Equal(t, `{ "custom": true }`, string(body), "Returns the expected body")
}
