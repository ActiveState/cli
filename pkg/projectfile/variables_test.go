package projectfile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestVariableStruct(t *testing.T) {
	variable := Variable{}
	dat := strings.TrimSpace(`
name: valueForName
value: valueForValue`)

	err := yaml.Unmarshal([]byte(dat), &variable)
	assert.Nil(t, err, "Should not throw an error")
	fail := variable.Parse()
	assert.NoError(t, fail.ToError(), "Should not fail")

	assert.Equal(t, "valueForName", variable.Name, "Name should be set")
	assert.NotNil(t, variable.Value.StaticValue, "Value should be set")
	assert.Equal(t, "valueForValue", *variable.Value.StaticValue, "Value should be set")
}

func TestVariableStruct_ComplexValue(t *testing.T) {
	variable := Variable{}
	dat := strings.TrimSpace(`
name: valueForName
value: 
    store: organization
    share: organization
`)

	err := yaml.Unmarshal([]byte(dat), &variable)
	assert.Nil(t, err, "Should not throw an error")
	fail := variable.Parse()
	assert.NoError(t, fail.ToError(), "Should not fail")

	assert.Equal(t, "valueForName", variable.Name, "Name should be set")
	assert.Nil(t, variable.Value.StaticValue, "Static Value should not be set")
	assert.NotNil(t, variable.Value.Store, "Store should be set")
	assert.Equal(t, "organization", string(*variable.Value.Store), "Store should be set")
	assert.NotNil(t, variable.Value.Share, "Share should be set")
	assert.Equal(t, "organization", string(*variable.Value.Share), "Share should be set")
}

func TestVariableStruct_ParseFail(t *testing.T) {
	variable := Variable{}
	dat := strings.TrimSpace(`
name: valueForName
value: 
    - notanobject
`)

	err := yaml.Unmarshal([]byte(dat), &variable)
	assert.Nil(t, err, "Should not throw an error")
	fail := variable.Parse()
	assert.Error(t, fail.ToError(), "Should fail")
	assert.True(t, fail.Type.Matches(FailParseVar), "Returns a FailParseVar failure")
}

func TestVariableStruct_ValidationFailStore(t *testing.T) {
	variable := Variable{}
	dat := strings.TrimSpace(`
name: valueForName
value: 
    store: notaValidValue
`)

	err := yaml.Unmarshal([]byte(dat), &variable)
	assert.Nil(t, err, "Should not throw an error")
	fail := variable.Parse()
	assert.Error(t, fail.ToError(), "Should fail")
	assert.True(t, fail.Type.Matches(FailValidateVarStore), "Returns a FailValidateVarStore failure")
}

func TestVariableStruct_ValidationFailShare(t *testing.T) {
	variable := Variable{}
	dat := strings.TrimSpace(`
name: valueForName
value: 
    share: notaValidValue
`)

	err := yaml.Unmarshal([]byte(dat), &variable)
	assert.Nil(t, err, "Should not throw an error")
	fail := variable.Parse()
	assert.Error(t, fail.ToError(), "Should fail")
	assert.True(t, fail.Type.Matches(FailValidateVarShare), "Returns a FailValidateVarShare failure")
}

func TestVariableStruct_ValidationFailStatic(t *testing.T) {
	// This error cannot happen due to yaml parsing, so we'll pretend we're a developer getting too creative
	variable := Variable{}
	staticValue := "foo"
	shareValue := VariableShare("organization")
	variable.Value.StaticValue = &staticValue
	variable.Value.Share = &shareValue
	fail := variable.Parse()

	assert.Error(t, fail.ToError(), "Should fail")
	assert.True(t, fail.Type.Matches(FailValidateStaticValueWithPull), "Returns a FailValidateStaticValueWithPull failure")
}
