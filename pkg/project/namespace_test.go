package project

import (
	"path/filepath"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/environment"
)

func newUUID(uuid string) *strfmt.UUID {
	u := strfmt.UUID(uuid)
	return &u
}

func TestParseNamespace(t *testing.T) {
	_, fail := ParseNamespace("valid/namespace")
	assert.NoError(t, fail.ToError(), "should parse a valid namespace")

	_, fail = ParseNamespace("valid/namespace#a10-b11c12-d13e14-f15")
	assert.NoError(t, fail.ToError(), "should parse a valid namespace with 'uuid'")

	_, fail = ParseNamespace("valid/namespace#")
	assert.NoError(t, fail.ToError(), "should parse a valid namespace with empty uuid")
}

func TestParseNamespace_Invalid(t *testing.T) {
	_, fail := ParseNamespace("invalid-namespace")
	assert.Error(t, fail.ToError(), "should get error with invalid namespace")
	assert.Equal(t, FailInvalidNamespace.Name, fail.Type.Name)

	_, fail = ParseNamespace("valid/namespace#invalidcommitid")
	assert.Error(t, fail.ToError(), "should get error with valid namespace and invalid commit id (basic hex and dash filter)")
	assert.Equal(t, FailInvalidNamespace.Name, fail.Type.Name)
}

func TestParseNamespaceOrConfigfile(t *testing.T) {
	rootpath, err := environment.GetRootPath()
	if err != nil {
		t.Fatal(err)
	}
	validConfigFile := filepath.Join(rootpath, "pkg", "projectfile", "testdata", "activestate.yaml")
	invalidConfigFile := filepath.Join(rootpath, "activestate.yaml.nope")

	var tests = []struct {
		name       string
		configFile string
		expected   *Namespaced
	}{
		{"InvalidConfigfile", invalidConfigFile, nil},
		{"FromConfigFile", validConfigFile, &Namespaced{Owner: "ActiveState", Project: "CodeIntel", CommitID: newUUID("00000000-0000-0000-0000-00000d7ebc72")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ns := NameSpaceForConfig(tt.configFile)
			assert.Equal(t, tt.expected, ns)
		})
	}
}
