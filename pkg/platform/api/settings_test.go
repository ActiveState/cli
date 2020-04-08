package api

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/stretchr/testify/assert"
)

func TestGetServiceURL(t *testing.T) {
	u := GetServiceURL(ServiceMono)
	assert.Equal(t, "https://platform.testing.tld"+constants.MonoAPIPath, u.String(), "Returns the expected service url")
}

func TestGetProjectHost(t *testing.T) {
	os.Setenv("ACTIVESTATE_API_HOST", "platform.activestate.com")
	defer func() {
		os.Unsetenv("ACTIVESTATE_API_TOKE")
	}()

	host := getProjectHost(ServiceMono)
	assert.Equal(t, "platform.activestate.com", *host)
}
