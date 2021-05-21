package inventory_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	inventoryMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func TestNew(t *testing.T) {
	mock := inventoryMock.Init()
	mock.MockPlatforms()
	defer mock.Close()

	client, _ := inventory.Init(authentication.Get())
	_, err := client.GetPlatforms(inventory_operations.NewGetPlatformsParams())
	assert.NoError(t, err)
}

func TestPersist(t *testing.T) {
	client := inventory.Get()
	client2 := inventory.Get()
	assert.True(t, client == client2, "Should return same pointer")
}
