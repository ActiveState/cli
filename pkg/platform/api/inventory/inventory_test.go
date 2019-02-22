package inventory_test

import (
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceInventory).String())
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/platforms")

	client := inventory.New()
	_, err := client.Platforms(inventory_operations.NewPlatformsParams())
	assert.NoError(t, err)
}

func TestPersist(t *testing.T) {
	client := inventory.Get()
	client2 := inventory.Get()
	assert.True(t, client == client2, "Should return same pointer")
}
