package analytics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const CatTest = "tests"

func TestSetup(t *testing.T) {
	setup()
	assert.NotNil(t, client, "Client is set")
}

func TestEvent(t *testing.T) {
	err := event(CatTest, "TestEvent")
	assert.NoError(t, err, "Should send event without causing an error")
}

func TestEventWithValue(t *testing.T) {
	err := eventWithValue(CatTest, "TestEventWithValue", 1)
	assert.NoError(t, err, "Should send event with value without causing an error")
}
