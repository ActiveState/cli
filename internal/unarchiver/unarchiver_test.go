package unarchiver_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/unarchiver"
)

type UnarchiverTestSuite struct {
	suite.Suite
}

func (suite *UnarchiverTestSuite) TestUnarchiveWithProgress() {
	// p := progress.New()
}

func (suite *UnarchiverTestSuite) TestUnarchive() {

	cases := []struct {
		name     string
		ua       unarchiver.Unarchiver
		testfile string
		prep     func(destination string)
	}{
		{
			"successful unpacking targz",
			unarchiver.NewTarGz(),
			"testfile.tar.gz", func(destination string) {
				err := os.WriteFile(destination, []byte{}, 0666)
				suite.Require().NoError(err)
			},
		},
		{
			"successful unpacking zip",
			unarchiver.NewZip(),
			"testfile.zip", func(destination string) {
				err := os.WriteFile(destination, []byte{}, 0666)
				suite.Require().NoError(err)
			},
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {

			testfile := filepath.Join("testdata", tc.testfile)

			tempDir, err := os.MkdirTemp("", "unarchiver-test-destination-root")
			suite.Require().NoError(err)
			destination := filepath.Join(tempDir, "destination")

			f, err := tc.ua.PrepareUnpacking(testfile, destination)
			suite.Assert().NoError(err)
			suite.Assert().NotNil(f)

			err = tc.ua.Unarchive(f, destination)
			suite.Assert().NoError(err)

			installedFiles, err := listFilesRecursively(destination)
			suite.Require().NoError(err)

			sort.Strings(installedFiles)

			suite.Assert().Len(installedFiles, 12)
		})
	}
}

func listFilesRecursively(dir string) ([]string, error) {
	res := make([]string, 0)
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == dir {
				return nil
			}
			res = append(res, info.Name())
			return nil
		})
	return res, err
}

func (suite *UnarchiverTestSuite) TestPrepareUnpackingWithError() {

	cases := []struct {
		name     string
		testfile string
		prep     func(destination string)
	}{{
		"cannot create destination", "testfile.tar.gz", func(destination string) {
			err := os.WriteFile(destination, []byte{}, 0666)
			suite.Require().NoError(err)
		},
	}, {
		"cannot open testfile", "non-existent.tar.gz", func(destination string) {},
	},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {

			ua := unarchiver.NewTarGz()
			testfile := filepath.Join("testdata", tc.testfile)

			tempDir, err := os.MkdirTemp("", "unarchiver-test-destination-root")
			suite.Require().NoError(err)
			destination := filepath.Join(tempDir, "destination")
			tc.prep(destination)

			f, err := ua.PrepareUnpacking(testfile, destination)
			suite.Assert().Nil(f)
			suite.Assert().Error(err)
		})
	}
}

func Test_UnarchiverTestSuite(t *testing.T) {
	suite.Run(t, new(UnarchiverTestSuite))
}
