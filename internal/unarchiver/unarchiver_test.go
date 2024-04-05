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

type mockCounter struct {
	Files     []string
	Count     int
	ByteCount int64
}

func (mc *mockCounter) Notify(fileName string, size int64, isDir bool) {
	if !isDir {
		mc.Count++
		mc.ByteCount += size
	}
	if fileName == "." {
		return
	}
	mc.Files = append(mc.Files, fileName)
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

			f, siz, err := tc.ua.PrepareUnpacking(testfile, destination)
			suite.Assert().NoError(err)
			suite.Assert().NotNil(f)
			suite.True(siz > int64(0))

			counter := &mockCounter{}

			tc.ua.SetNotifier(counter.Notify)

			err = tc.ua.Unarchive(f, siz, destination)
			suite.Assert().NoError(err)

			suite.Assert().Equal(9, counter.Count, "nine files unpacked")
			// For this example the byte count will be very low, but maybe OS / file system dependent on how big exactly, so we just compare to zero
			suite.True(counter.ByteCount > int64(0))

			installedFiles, err := listFilesRecursively(destination)
			suite.Require().NoError(err)

			sort.Strings(installedFiles)
			sort.Strings(counter.Files)

			suite.Assert().Len(installedFiles, 12)
			suite.Assert().Equal(installedFiles, counter.Files)
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

			f, siz, err := ua.PrepareUnpacking(testfile, destination)
			suite.Assert().Nil(f)
			suite.Assert().Zero(siz)
			suite.Assert().Error(err)
		})
	}
}

func Test_UnarchiverTestSuite(t *testing.T) {
	suite.Run(t, new(UnarchiverTestSuite))
}
