package strutils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTemplate(t *testing.T) {
	result, err := ParseTemplate("{{eq .Foo .Bar}}", map[string]string{"Foo": "foo", "Bar": "bar"}, nil)
	require.NoError(t, err)
	require.Equal(t, "", result)
}
