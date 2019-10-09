package api_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/stretchr/testify/assert"
)

func TestGetServiceURL(t *testing.T) {
	u := api.GetServiceURL(api.ServiceMono)
	assert.Equal(t, "https://platform.testing.tld"+constants.MonoAPIPath, u.String(), "Returns the expected service url")
}
