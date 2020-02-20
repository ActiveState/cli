package api

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/stretchr/testify/assert"
)

func TestGetServiceURL(t *testing.T) {
	u := GetServiceURL(ServiceMono)
	assert.Equal(t, "https://platform.testing.tld"+constants.MonoAPIPath, u.String(), "Returns the expected service url")
}

func TestGetProjectHost(t *testing.T) {
	host := getProjectHost()
	assert.Equal(t, "platform.activestate.com", *host)
}
