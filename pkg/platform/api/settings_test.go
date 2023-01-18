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
	os.Setenv(constants.APIHostEnvVarName, constants.DefaultAPIHost)
	defer func() {
		os.Unsetenv(constants.APIHostEnvVarName)
	}()

	host := getProjectHost(ServiceMono)
	assert.Equal(t, constants.DefaultAPIHost, *host)
}
