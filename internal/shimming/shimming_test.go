package shimming_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/shimming"
)

type ShimmingTestSuite struct {
	suite.Suite
	testdir string
}

func (suite *ShimmingTestSuite) BeforeTest(suiteName, testName string) {
	root := environment.GetRootPathUnsafe()
	suite.testdir = filepath.Join(root, "internal", "shimming", "testdata")
}

func (suite *ShimmingTestSuite) TestBinariesToShim() {
	shim := shimming.NewShim([]string{"binary1", "binary2", "binarynotexist"})
	binaries := shim.BinariesToShim([]string{suite.testdir})
	suite.Contains(binaries, "binary1")
	suite.Contains(binaries, "binary2")
	suite.NotContains(binaries, "binarynotexist")
}

func (suite *ShimmingTestSuite) TestCollectionBinariesToShim() {
	shim1 := shimming.NewShim([]string{"binary1", "binary2", "binarynotexist"})
	shim2 := shimming.NewShim([]string{"binary2", "binarynotexist"})
	collection := shimming.NewCollection()
	collection.RegisterShim(shim1)
	collection.RegisterShim(shim2)

	binaries := collection.BinariesToShim([]string{suite.testdir})
	suite.Contains(binaries, "binary1")
	suite.Contains(binaries, "binary2")
	suite.NotContains(binaries, "binarynotexist")
}

func (suite *ShimmingTestSuite) TestShimBinaries() {
	shim1 := shimming.NewShim([]string{"binary1"})
	collection := shimming.NewCollection()
	collection.RegisterShim(shim1)

	dir, fail := collection.ShimBinaries([]string{suite.testdir})
	suite.Require().NoError(fail.ToError())

	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".bat"
	}
	suite.FileExists(filepath.Join(dir, "binary1"+suffix))
}

func (suite *ShimmingTestSuite) TestShimBinariesEmpty() {
	collection := shimming.NewCollection()

	dir, fail := collection.ShimBinaries([]string{suite.testdir})
	suite.Require().NoError(fail.ToError())

	suite.Empty(dir)
}

func TestShimmingTestSuite(t *testing.T) {
	suite.Run(t, new(ShimmingTestSuite))
}
