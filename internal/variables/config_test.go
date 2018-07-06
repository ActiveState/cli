package variables

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestConfigVariables(t *testing.T) {
	config := []configVariable{}
	viper.Set("variables", config) // clear
	err := viper.UnmarshalKey("variables", &config)
	assert.NoError(t, err, "Unmarshalled no variables")
	assert.Equal(t, 0, len(config), "No variables should be defined")

	testValue = "bar"
	value := ConfigValue("foo", "baz")
	assert.Equal(t, testValue, value, "Prompt result returned")
	err = viper.UnmarshalKey("variables", &config)
	assert.NoError(t, err, "Unmarshalled saved variable")
	assert.Equal(t, 1, len(config), "One saved variable")
	assert.Equal(t, "foo", config[0].Name, "Variable name stored correctly")
	assert.Equal(t, "bar", config[0].Value, "Variable value stored correctly")
	assert.Equal(t, "baz", config[0].Project, "Variable project stored correctly")
	testValue = "" // reset

	value = ConfigValue("foo", "baz")
	assert.Equal(t, "bar", value, "Looked up stored value")
}
