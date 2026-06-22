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
		name      string
		ua        unarchiver.Unarchiver
		testfile  string
		wantErr   bool
		wantFiles int
	}{
		{
			// testfile.tar.gz is fully contained.
			"successful tar.gz unpacking",
			unarchiver.NewTarGz(),
			"testfile.tar.gz",
			false,
			11,
		},
		{
			// testfile-escapes.tar.gz has a root-level symlink (symlink-to-file3 ->
			// ../b/c/file3) whose target resolves outside the destination, so it is
			// rejected when treated as untrusted.
			"escaping tar.gz rejected when untrusted",
			unarchiver.NewTarGz(unarchiver.WithUntrustedSource()),
			"testfile-escapes.tar.gz",
			true,
			0,
		},
		{
			// When trusted (the default), the same archive extracts as before
			// (Platform artifacts may legitimately link outside the destination).
			"escaping tar.gz extracts when trusted",
			unarchiver.NewTarGz(),
			"testfile-escapes.tar.gz",
			false,
			12,
		},
		{
			// The zip fixture stores its symlinks as ordinary files, so every entry is
			// contained and extraction succeeds.
			"successful zip unpacking",
			unarchiver.NewZip(),
			"testfile.zip",
			false,
			12,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {

			testfile := filepath.Join("testdata", tc.testfile)

			tempDir, err := os.MkdirTemp("", "unarchiver-test-destination-root")
			suite.Require().NoError(err)
			destination := filepath.Join(tempDir, "destination")

			f, err := tc.ua.PrepareUnpacking(testfile, destination)
			suite.Require().NoError(err)
			suite.Require().NotNil(f)

			err = tc.ua.Unarchive(f, destination)
			if tc.wantErr {
				suite.Assert().Error(err)
				return
			}
			suite.Assert().NoError(err)

			installedFiles, err := listFilesRecursively(destination)
			suite.Require().NoError(err)

			sort.Strings(installedFiles)

			suite.Assert().Len(installedFiles, tc.wantFiles)
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
