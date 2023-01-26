package request

import (
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUsage(t *testing.T) {
	// This is not a test meant to run on CI, but rather we can use to debug things locally.
	// To run this comment out the next line and comment out the `if condition.InUnitTest() {` section inside api/settings.go
	t.Skip("For development use only")

	tests := []struct {
		name           string
		organizationID strfmt.UUID
	}{
		{
			"Basic",
			strfmt.UUID("5587a9d9-a564-4d49-89fd-4dd04d51f13e"), // ActiveState-labs
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			auth := authentication.LegacyGet()
			require.NoError(t, auth.Authenticate())

			client := graphql.New()
			client.SetTokenProvider(auth)

			resp := model.RuntimeUsageResponse{}
			err := client.Run(RuntimeUsage(tt.organizationID), &resp)

			usage := resp.Usage[0]

			require.NoError(t, err)
			require.NotEmpty(t, resp.Usage)
			require.NotEmpty(t, usage.ActiveRuntimesDynamicAndStatic)

		})
	}
}
