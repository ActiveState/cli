package fileutils

import (
	"bufio"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(oldPath, newPath, filler string, binary bool) []byte {
	l := 1000000
	res := make([]byte, l)
	for i := 0; i < l; i++ {
		if binary {
			res[i] = byte(rand.Intn(256))
		} else {
			res[i] = byte(rand.Intn(255) + 1)
		}
	}

	return res
}

var result []byte

// BenchmarkRead compares how fast it takes to read a file that we want to replace later
// Three methods are compared:
// 1) read everything in bulk
// 2) use bufio.NewReader to read the data
// 3) read only a fixed amount of data
// On my SSD, (3) is about 15% faster than (1), (2) is the slowest by a factor of 8 compared to (3)
// Streaming would therefore improve the performance, but is more complicated to implement and leads to a slower replacement step
func BenchmarkRead(b *testing.B) {
	oldPath := "abc/def/ghi"
	newPath := "def/ghi"
	byts := setup(oldPath, newPath, "/bin/python.sh", true)

	testFile := TempFileUnsafe("", "")
	_, err := testFile.Write(byts)
	if err != nil {
		b.Errorf("failed to write test file: %v", err)
	}
	err = testFile.Close()
	if err != nil {
		b.Errorf("failed to close test file: %v", err)
	}
	defer os.Remove(testFile.Name())

	b.ResetTimer()

	b.Run("read file (bulk)", func(bb *testing.B) {
		for n := 0; n < bb.N; n++ {
			f, err := os.Open(testFile.Name())
			if err != nil {
				bb.Errorf("Failed to open file: %v", err)
			}
			defer func() {
				f.Close()
			}()
			r, err := ioutil.ReadAll(f)
			if err != nil {
				bb.Errorf("Received error reading: %v", err)
			}
			if len(r) != len(byts) {
				bb.Errorf("Expected to read %d bytes, read = %d", len(byts), len(r))
			}
		}
	})

	b.Run("read file (bufio)", func(bb *testing.B) {
		for n := 0; n < bb.N; n++ {
			f, err := os.Open(testFile.Name())
			if err != nil {
				bb.Errorf("Failed to open file: %v", err)
			}
			br := bufio.NewReaderSize(f, 1024)
			defer func() {
				f.Close()
			}()
			var nr int
			for {
				_, err := br.ReadByte()
				if err == io.EOF {
					break
				}
				nr++
			}
			if err != nil {
				bb.Errorf("Received error reading: %v", err)
			}
			if nr != len(byts) {
				bb.Errorf("Expected to read %d bytes, read = %d", len(byts), nr)
			}
		}
	})

	b.Run("read file (stream)", func(bb *testing.B) {
		for n := 0; n < bb.N; n++ {
			f, err := os.Open(testFile.Name())
			if err != nil {
				bb.Errorf("Failed to open file: %v", err)
			}
			defer func() {
				f.Close()
			}()
			b := make([]byte, 1024)
			var nr int
			for {
				n, err := f.Read(b)
				if n == 0 && err == io.EOF {
					break
				}
				nr += n
			}
			if err != nil {
				bb.Errorf("Received error reading: %v", err)
			}
			if nr != len(byts) {
				bb.Errorf("Expected to read %d bytes, read = %d", len(byts), nr)
			}
		}
	})
}

// BenchmarkWrite measures the time it safes to write the replaced file back to disk
// The measurements should be compared to the time it takes to read file contents and to replace the strings.
// On my SSD Drive, writing to disk takes about 1.6 times as long as the replacement step
// and approximately as long as reading the file contents to memory initially.
func BenchmarkWrite(b *testing.B) {
	oldPath := "abc/def/ghi"
	newPath := "def/ghi"
	byts := setup(oldPath, newPath, "/bin/python.sh", true)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		f := TempFileUnsafe("", "")
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()
		nw, err := f.Write(byts)
		if err != nil {
			b.Errorf("Received error writing: %v", err)
		}
		if nw != len(byts) {
			b.Errorf("Expected to write %d bytes, written = %d", len(byts), nw)
		}
	}
}

// BenchmarkReplace measures how long it takes to replace text in large files
// Two main methods are compared (one based on regular expressions, the other is more specialized)
// Replacing in binary files (with nul-terminated strings) and text files is considered separately.
// Sample results are:
// BenchmarkReplace/with_regex_(binary)-8              2335            458652 ns/op         2021164 B/op         35 allocs/op
// BenchmarkReplace/without_regex_(binary)-8                  12064             96341 ns/op               4 B/op          1 allocs/op
// BenchmarkReplace/with_regex_(string)-8                      2142            470041 ns/op         2019299 B/op         28 allocs/op
// BenchmarkReplace/without_regex_(string)-8                   4059            257515 ns/op         1007616 B/op          1 allocs/op
func BenchmarkReplace(b *testing.B) {
	oldPath := "abc/def/ghi"
	newPath := "def/ghi"
	binByts := setup(oldPath, newPath, "/bin/python.sh", true)
	stringByts := setup(oldPath, newPath, "/bin/python.sh", false)
	runs := []struct {
		name  string
		f     func([]byte, string, string) (bool, []byte, error)
		input []byte
	}{
		{
			"with regex (binary)",
			replaceInFile,
			binByts,
		},
		{
			"with regex (string)",
			replaceInFile,
			stringByts,
		},
	}
	b.ResetTimer()

	for _, run := range runs {
		b.Run(run.name, func(bb *testing.B) {
			var r []byte
			for n := 0; n < bb.N; n++ {
				_, res, err := run.f(run.input, oldPath, newPath)
				if err != nil {
					bb.Errorf("Received error: %v", err)
				}
				if len(res) != len(run.input) {
					bb.Errorf("Expected len = %d, got = %d", len(run.input), len(res))
				}
				r = res
			}
			result = r

		})
	}
}

func TestReplaceBytesError(t *testing.T) {
	b := []byte("Hello world\x00")
	_, _, err := replaceInFile(b, "short", "longer")
	assert.Error(t, err)
}

func TestReplaceBytes(t *testing.T) {
	oldPath := "abc/def/ghi"
	newPath := "def/ghi"

	byts := []byte("123abc/def/ghi/bin/python\x00456abc/def/ghi/bin/perl\x00other")
	expected := []byte("123def/ghi/bin/python\x00\x00\x00\x00\x00456def/ghi/bin/perl\x00\x00\x00\x00\x00other")

	text := []byte("'123abc/def/ghi/bin/python'456'abc/def/ghi/bin/perl'other")
	textExpected := []byte("'123def/ghi/bin/python'456'def/ghi/bin/perl'other")

	noMatchByts := []byte("nothing to match here\x00")
	noMatchText := []byte("nothing to match here\x00")

	runs := []struct {
		name     string
		f        func([]byte, string, string) (bool, []byte, error)
		input    []byte
		expected []byte
		changes  bool
	}{
		{"nul-terminated with regex", replaceInFile, byts, expected, true},
		{"text with regex", replaceInFile, text, textExpected, true},
		{"nul-terminated with regex - no match", replaceInFile, noMatchByts, noMatchByts, false},
		{"text with regex - no match", replaceInFile, noMatchText, noMatchText, false},
	}

	for _, run := range runs {
		t.Run(run.name, func(tt *testing.T) {

			count, res, err := run.f(run.input, oldPath, newPath)
			require.NoError(tt, err)

			assert.Equal(t, run.expected, res)
			assert.Equal(t, run.changes, count)
		})
	}
}
