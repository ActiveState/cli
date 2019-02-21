package api_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/stretchr/testify/assert"
)

func TestGetServiceURL(t *testing.T) {
	u := api.GetServiceURL(api.ServicePlatform)
	assert.Equal(t, "https://testing.tld"+constants.PlatformAPIPath, u.String(), "Returns the expected service url")
}

func TestGetSettings(t *testing.T) {
	s := api.GetSettings(api.ServicePlatform)
	assert.Equal(t, "testing.tld", s.Host)
}
