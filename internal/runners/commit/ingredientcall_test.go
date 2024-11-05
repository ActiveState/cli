package commit

import (
	"sort"
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
	runtime_deps = [
		Req(name="python", namespace="language", version=Eq(value="3.7.10"))
	]
)
`

const simpleAlteredScript = `
main = ingredient(
	src = ["*/**.py", "pyproject.toml"],
	hash = "<old hash>",
	runtime_deps = [
		Req(name="python", namespace="language", version=Eq(value="3.7.10")),
		Req(name="python-module-builder", namespace="builder", version=Gt(value="0")),
	]
)
`

const depTypesScript = `
main = ingredient(
	src = ["*/**.py", "pyproject.toml"],
	hash = "<old hash>",
	runtime_deps = [
		Req(name="runtimedep", namespace="language", version=Eq(value="1.0"))
	],
	build_deps = [
		Req(name="builddep", namespace="language", version=Eq(value="2.0"))
	],
	test_deps = [
		Req(name="testdep", namespace="language", version=Eq(value="3.0"))
	],
)
`

const invalidDepsScript = `
main = ingredient(
	runtime_deps = "I should be a slice"
)
`

const invalidDepScript = `
main = ingredient(
	runtime_deps = [ 
		Req(name="runtimedep", namespace="language", version=Eq(value="1.0")),
		"I should be a Req" 
	]
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
			"6a7c7bd03f10e832",
		},
		{
			"Simple Altered",
			args{
				script: simpleAlteredScript,
				seed:   "",
			},
			"1471d1796a57e938",
		},
		{
			"Simple With Seed",
			args{
				script: simpleScript,
				seed:   "seed",
			},
			"a9c1a37b5dd6f0d6",
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
						VersionRequirements: "==3.7.10",
						Type:                request.DependencyTypeRuntime,
					},
					Conditions: []request.Dependency{},
				},
			},
			assert.NoError,
		},
		{
			"All Types",
			depTypesScript,
			[]request.PublishVariableDep{
				{
					Dependency: request.Dependency{
						Name:                "runtimedep",
						Namespace:           "language",
						VersionRequirements: "==1.0",
						Type:                request.DependencyTypeRuntime,
					},
					Conditions: []request.Dependency{},
				},
				{
					Dependency: request.Dependency{
						Name:                "builddep",
						Namespace:           "language",
						VersionRequirements: "==2.0",
						Type:                request.DependencyTypeBuild,
					},
					Conditions: []request.Dependency{},
				},
				{
					Dependency: request.Dependency{
						Name:                "testdep",
						Namespace:           "language",
						VersionRequirements: "==3.0",
						Type:                request.DependencyTypeTest,
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
				assert.Error(t, err)
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
			if !tt.wantErr(t, err, "") {
				return
			}
			sort.Slice(tt.want, func(i, j int) bool {
				return tt.want[i].Name < tt.want[j].Name
			})
			sort.Slice(got, func(i, j int) bool {
				return got[i].Name < got[j].Name
			})
			assert.Equalf(t, tt.want, got, "resolveDependencies()")
		})
	}
}
