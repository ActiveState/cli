package inventory_test

import (
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"

	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	inventoryMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	mock := inventoryMock.Init()
	mock.MockPlatforms()
	defer mock.Close()

	client := inventory.Init()
	_, err := client.Platforms(inventory_operations.NewPlatformsParams())
	assert.NoError(t, err)
}

func TestPersist(t *testing.T) {
	client := inventory.Get()
	client2 := inventory.Get()
	assert.True(t, client == client2, "Should return same pointer")
}
