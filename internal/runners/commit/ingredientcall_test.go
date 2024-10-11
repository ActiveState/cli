package commit

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const simpleScript = `
main = ingredient(
	src = ["*/**.py", "pyproject.toml"],
	hash = "<old hash>",
	deps = [
		Req(name="python", namespace="language", version=Eq(value="3.7.10"))
	]
)
`

const simpleAlteredScript = `
main = ingredient(
	src = ["*/**.py", "pyproject.toml"],
	hash = "<old hash>",
	deps = [
		Req(name="python", namespace="language", version=Eq(value="3.7.10")),
		Req(name="python-module-builder", namespace="builder", version=Gt(value="0")),
	]
)
`

const invalidDepsScript = `
main = ingredient(
	deps = "I should be a slice"
)
`

const invalidDepScript = `
main = ingredient(
	deps = [ "I should be a Req" ]
)
`

func Test_hashFuncCall(t *testing.T) {

	type args struct {
		script string
		seed   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Simple",
			args{
				script: simpleScript,
				seed:   "",
			},
			"6fa602bc516a918e",
		},
		{
			"Simple Altered",
			args{
				script: simpleAlteredScript,
				seed:   "",
			},
			"b74d9b5cf2e6b0ee",
		},
		{
			"Simple With Seed",
			args{
				script: simpleScript,
				seed:   "seed",
			},
			"9a9915a8bf84c7ad",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs, err := buildscript.Unmarshal([]byte(tt.args.script))
			require.NoError(t, err)
			fc := bs.FunctionCalls("ingredient")[0]
			hashBefore := fc.Argument("hash").(string)
			got, err := hashFuncCall(fc, tt.args.seed)
			if err != nil {
				t.Errorf("hashFuncCall() error = %v", err)
				return
			}
			hashAfter := fc.Argument("hash").(string)
			if got != tt.want {
				t.Errorf("hashFuncCall() got = %v, want %v", got, tt.want)
			}
			assert.Equal(t, hashBefore, hashAfter) // calculating the hash should not affect the hash
		})
	}
}

func TestIngredientCall_resolveDependencies(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		want    []request.PublishVariableDep
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"Simple",
			simpleScript,
			[]request.PublishVariableDep{
				{
					Dependency: request.Dependency{
						Name:                "python",
						Namespace:           "language",
						VersionRequirements: "3.7.10",
					},
					Conditions: []request.Dependency{},
				},
			},
			assert.NoError,
		},
		{
			"Invalid Deps",
			invalidDepsScript,
			nil,
			func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.ErrorAs(t, err, &invalidDepsValueType{})
			},
		},
		{
			"Invalid Dep",
			invalidDepScript,
			nil,
			func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.ErrorAs(t, err, &invalidDepValueType{})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs, err := buildscript.Unmarshal([]byte(tt.script))
			require.NoError(t, err)
			fc := bs.FunctionCalls("ingredient")[0]
			i := &IngredientCall{script: bs, funcCall: fc}
			got, err := i.resolveDependencies()
			if !tt.wantErr(t, err, fmt.Sprintf("resolveDependencies()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "resolveDependencies()")
		})
	}
}
