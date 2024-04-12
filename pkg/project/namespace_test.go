package project

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/environment"
)

func TestParseNamespace(t *testing.T) {
	_, err := ParseNamespace("valid/namespace")
	assert.NoError(t, err, "should parse a valid namespace")

	_, err = ParseNamespace("valid/namespace#00000000-0000-0000-0000-000000000000")
	assert.NoError(t, err, "should parse a valid namespace with 'uuid'")

	_, err = ParseNamespace("valid/namespace#a10-b11c12-d13e14-f15")
	assert.Error(t, err, "should fail to parse an invalid 'uuid'")

	_, err = ParseNamespace("valid/namespace#")
	assert.NoError(t, err, "should parse a valid namespace with empty uuid")
}

func TestParseNamespace_Invalid(t *testing.T) {
	_, err := ParseNamespace("invalid-namespace")
	assert.Error(t, err, "should get error with invalid namespace")

	_, err = ParseNamespace("valid/namespace#invalidcommitid")
	assert.Error(t, err, "should get error with valid namespace and invalid commit id (basic hex and dash filter)")
}

func TestParseProjectNoOwner(t *testing.T) {
	parsed, err := ParseProjectNoOwner("project")
	assert.NoError(t, err, "should be able to parse project part of namspace")
	assert.Empty(t, parsed.Owner)
	assert.Equal(t, parsed.Project, "project")
	assert.Empty(t, parsed.CommitID)
	assert.True(t, parsed.AllowOmitOwner)
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
		{"FromConfigFile", validConfigFile, &Namespaced{Owner: "ActiveState", Project: "CodeIntel"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ns := NameSpaceForConfig(tt.configFile)
			assert.Equal(t, tt.expected, ns)
		})
	}
}
