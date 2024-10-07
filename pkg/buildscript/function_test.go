package buildscript_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/buildscript/processors/ingredient"
	"github.com/stretchr/testify/require"
)

func TestMarshalBEIngredientAndReqFunc(t *testing.T) {
	bs, err := buildscript.Unmarshal([]byte(`
main = ingredient( 
	name = "pytorch",
	src = ["*/**.py", "pyproject.toml"],
	deps = [
		Req(name="python", version=Eq(value="3.7.10"))
	]
)
`))
	require.NoError(t, err)

	marshaller := ingredient.NewProcessor(nil)
	buildscript.RegisterFunctionProcessor("ingredient", marshaller.MarshalBuildExpression)

	data, err := bs.MarshalBuildExpression()
	require.NoError(t, err)

	result := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(data, &result))

	hash := getKey(t, result, "let", "in", "ingredient", "hash")
	require.NotEmpty(t, hash)

	_ = data
}

func getKey(t *testing.T, data map[string]interface{}, keys ...string) any {
	var next any
	var ok bool
	for i, key := range keys {
		next, ok = data[key]
		if !ok {
			t.Fatalf("key %s not found in data", strings.Join(keys[0:i+1], "."))
		}
		if len(keys) > i+1 {
			if data, ok = next.(map[string]interface{}); !ok {
				t.Fatalf("key %s has non-map value: '%v'", strings.Join(keys[0:i+1], "."), next)
			}
		}
	}
	return next
}
