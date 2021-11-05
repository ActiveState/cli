package test

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewClient(t *testing.T) {
	teardown := setup()
	defer teardown()

	assert.NotNil(t, client)
}
