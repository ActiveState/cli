package envdef_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
)

type EnvironmentTestSuite struct {
	suite.Suite
}

func (suite *EnvironmentTestSuite) TestMergeVariables() {

	ev1 := envdef.EnvironmentVariable{}
	err := json.Unmarshal([]byte(`{
		"env_name": "V",
		"values": ["a", "b"]
		}`), &ev1)
	require.NoError(suite.T(), err)
	ev2 := envdef.EnvironmentVariable{}
	err = json.Unmarshal([]byte(`{
		"env_name": "V",
		"values": ["c", "d"]
		}`), &ev2)
	require.NoError(suite.T(), err)

	expected := &envdef.EnvironmentVariable{}
	err = json.Unmarshal([]byte(`{
		"env_name": "V",
		"values": ["c", "d", "a", "b"],
		"join": "prepend"
		}`), expected)
	require.NoError(suite.T(), err)

	suite.Assert().True(expected.Inherit, "inherit should be true")
	suite.Assert().Equal(":", expected.Separator)

	res, err := ev1.Merge(ev2)
	suite.Assert().NoError(err)
	suite.Assert().Equal(expected, res)
}

func (suite *EnvironmentTestSuite) TestMerge() {
	ed1 := &envdef.EnvironmentDefinition{}

	err := json.Unmarshal([]byte(`{
			"env": [{"env_name": "V", "values": ["a", "b"]}],
			"installdir": "abc"
		}`), ed1)
	require.NoError(suite.T(), err)

	ed2 := envdef.EnvironmentDefinition{}
	err = json.Unmarshal([]byte(`{
			"env": [{"env_name": "V", "values": ["c", "d"]}],
			"installdir": "abc"
		}`), &ed2)
	require.NoError(suite.T(), err)

	expected := envdef.EnvironmentDefinition{}
	err = json.Unmarshal([]byte(`{
			"env": [{"env_name": "V", "values": ["c", "d", "a", "b"]}],
			"installdir": "abc"
		}`), &expected)
	require.NoError(suite.T(), err)

	ed1, err = ed1.Merge(&ed2)
	suite.Assert().NoError(err)
	require.NotNil(suite.T(), ed1)
	suite.Assert().Equal(expected, *ed1)
}

func (suite *EnvironmentTestSuite) TestInheritPath() {
	ed1 := &envdef.EnvironmentDefinition{}

	err := json.Unmarshal([]byte(`{
			"env": [{"env_name": "PATH", "values": ["NEWVALUE"]}],
			"join": "prepend",
			"inherit": true,
			"separator": ":"
		}`), ed1)
	require.NoError(suite.T(), err)

	env, err := ed1.GetEnvBasedOn(func(k string) (string, bool) {
		return "OLDVALUE", true
	})
	require.NoError(suite.T(), err)
	suite.True(strings.HasPrefix(env["PATH"], "NEWVALUE"), "%s does not start with NEWVALUE", env["PATH"])
	suite.True(strings.HasSuffix(env["PATH"], "OLDVALUE"), "%s does not end with OLDVALUE", env["PATH"])
}

func (suite *EnvironmentTestSuite) TestSharedTests() {

	type testCase struct {
		Name        string                         `json:"name"`
		Definitions []envdef.EnvironmentDefinition `json:"definitions"`
		BaseEnv     map[string]string              `json:"base_env"`
		Expected    map[string]string              `json:"result"`
		IsError     bool                           `json:"error"`
	}

	td, err := ioutil.ReadFile("runtime_test_cases.json")
	require.NoError(suite.T(), err)

	cases := &[]testCase{}

	err = json.Unmarshal(td, cases)
	require.NoError(suite.T(), err, "unmarshal the test cases")

	for _, tc := range *cases {
		suite.Run(tc.Name, func() {
			ed := &tc.Definitions[0]
			for i, med := range tc.Definitions[1:] {
				ed, err = ed.Merge(&med)
				if tc.IsError {
					suite.Assert().Error(err)
					return
				}
				suite.Assert().NoError(err, "error merging %d-th definition", i)
			}

			lookupEnv := func(k string) (string, bool) {
				res, ok := tc.BaseEnv[k]
				return res, ok
			}

			res, err := ed.GetEnvBasedOn(lookupEnv)
			if tc.IsError {
				suite.Assert().Error(err)
				return
			}
			suite.Assert().NoError(err)
			suite.Assert().Equal(tc.Expected, res)
		})
	}

}

func (suite *EnvironmentTestSuite) TestValueString() {
	ev1 := envdef.EnvironmentVariable{}
	err := json.Unmarshal([]byte(`{
		"env_name": "V",
		"values": ["a", "b"]
		}`), &ev1)
	require.NoError(suite.T(), err)

	res := ev1.ValueString()
	suite.Assert().Equal("a:b", res)
}

func (suite *EnvironmentTestSuite) TestGetEnv() {
	ed1 := envdef.EnvironmentDefinition{}
	err := json.Unmarshal([]byte(`{
			"env": [{"env_name": "V", "values": ["a", "b"]}],
			"installdir": "abc"
		}`), &ed1)
	require.NoError(suite.T(), err)

	res := ed1.GetEnv(true)
	suite.Assert().Equal(map[string]string{
		"V": "a:b",
	}, res)
}

func (suite *EnvironmentTestSuite) TestFindBinPathFor() {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(suite.T(), err, "creating temporary directory")
	defer os.RemoveAll(tmpDir)

	ed1 := envdef.EnvironmentDefinition{}
	err = json.Unmarshal([]byte(`{
			"env": [{"env_name": "PATH", "values": ["${INSTALLDIR}/bin", "${INSTALLDIR}/bin2"]}],
			"installdir": "abc"
		}`), &ed1)
	require.NoError(suite.T(), err, "un-marshaling test json blob")

	constants := envdef.NewConstants(tmpDir)
	// expand variables
	ed1.ExpandVariables(constants)

	suite.Assert().Equal("", ed1.FindBinPathFor("executable"), "executable should not exist")

	err = fileutils.Touch(filepath.Join(tmpDir, "bin2", "executable"))
	require.NoError(suite.T(), err, "creating dummy file")
	suite.Assert().Equal(filepath.Join(tmpDir, "bin2"), ed1.FindBinPathFor("executable"), "executable should be found")
}

func TestEnvironmentTestSuite(t *testing.T) {
	suite.Run(t, new(EnvironmentTestSuite))
}
