package projectfile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestParseShorthandValid(t *testing.T) {
	pl, err := parseData(pFileYAMLValid.asLongYAML(), "junk/path")
	require.NoError(t, err, "Parse longhand file without error")
	require.NotEmpty(t, pl.Constants, "Longhand constants are not empty")
	for _, c := range pl.Constants {
		require.NotEmpty(t, c.Name, "Name field of (longhand) constant is not empty")
		require.NotEmpty(t, c.Value, "Value field of (longhand) constant is not empty")
	}

	ps, err := parseData(pFileYAMLValid.asShortYAML(), "junk/path")
	require.NoError(t, err, "Parse shorthand file without error")
	require.NotEmpty(t, ps.Constants, "Shorthand constants are not empty")
	for _, c := range ps.Constants {
		require.NotEmpty(t, c.Name, "Name field of (shorthand) constant is not empty")
		require.NotEmpty(t, c.Value, "Value field of (shorthand) constant is not empty")
	}

	require.Equal(t, pl.Constants, ps.Constants, "Longhand constants slice is equal to shorthand constants slice")
}

func TestParseShorthandBadData(t *testing.T) {
	tests := []struct {
		name     string
		fileData pFileYAML
	}{
		{
			"array in name",
			pFileYAML{`["test", "array", "name"]`, `valid`},
		},
		{
			"array in value",
			pFileYAML{`valid`, `["test", "array", "value"]`},
		},
		{
			"new field in name",
			pFileYAML{`- 42`, `valid`},
		},
		{
			"new field in value",
			pFileYAML{`valid`, `- 42`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			longYAML := tt.fileData.asLongYAML()
			_, err := parseData(longYAML, "junk/path")
			require.Error(t, err, "Parse bad longhand yaml with failure")

			shortYAML := tt.fileData.asShortYAML()
			_, shErr := parseData(shortYAML, "junk/path")
			require.Error(t, shErr, "Parse bad shorthand yaml with failure")
		})
	}
}

type CustomFields struct {
	Desc   string `yaml:"desc,omitempty"`
	Truthy bool   `yaml:"truthy,omitempty"`
}

type Custom struct {
	NameVal      `yaml:",inline"`
	CustomFields `yaml:",inline"`
}

func (c *Custom) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&c.NameVal); err != nil {
		return err
	}
	if err := unmarshal(&c.CustomFields); err != nil {
		return err
	}
	return nil
}

func TestParseShorthandSimpleStruct(t *testing.T) {
	c := Custom{}
	data := strings.TrimSpace(`
name: valueForName
value: valueForValue
desc: valueForDesc
truthy: true
` + "\n")

	err := yaml.Unmarshal([]byte(data), &c)
	require.NoError(t, err, "Unmarshal without error")

	assert.Equal(t, "valueForName", c.Name, "Name (longhand) should be set properly")
	assert.Equal(t, "valueForValue", c.Value, "Value (longhand) should be set properly")
	assert.Equal(t, "valueForDesc", c.Desc, "Desc (longhand) should be set properly")
	assert.Equal(t, true, c.Truthy, "Truthy (longhand) should be set true")

	c = Custom{}
	data = strings.TrimSpace(`
valueForName: valueForValue
` + "\n")

	err = yaml.Unmarshal([]byte(data), &c)
	require.NoError(t, err, "Unmarshal without error")

	assert.Equal(t, "valueForName", c.Name, "Name (shorthand) should be set properly")
	assert.Equal(t, "valueForValue", c.Value, "Value (shorthand) should be set properly")
}
