package logging

import (
	"io/fs"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilePathForCmd(t *testing.T) {
	filename := FileNameForCmd("cmd-name", 123)
	path := FilePathForCmd("cmd-name", 123)
	assert.NotEqual(t, filename, path)
	assert.True(t, strings.Contains(path, "cmd-name-123"))
}

type mockFile struct {
	name    string
	modTime time.Time
}

func (m mockFile) Name() string       { return m.name }
func (m mockFile) Size() int64        { return -1 }
func (m mockFile) Mode() fs.FileMode  { return os.ModePerm }
func (m mockFile) ModTime() time.Time { return m.modTime }
func (m mockFile) IsDir() bool        { return false }
func (m mockFile) Sys() interface{}   { return nil }

func Test_rotateLogs(t *testing.T) {
	type args struct {
		files        []fs.FileInfo
		timeCutoff   time.Time
		amountCutoff int
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"Rotate 2 files based on cutoff",
			args{
				[]fs.FileInfo{
					&mockFile{"prefixA-123-expired1.log", time.Now().Add(-time.Hour * 2)},
					&mockFile{"prefixA-123-notexpired1.log", time.Now()},
					&mockFile{"prefixA-123-expired2.log", time.Now().Add(-time.Hour * 2)},
					&mockFile{"prefixA-123-expired3.log", time.Now().Add(-time.Hour * 2)},
					&mockFile{"prefixA-123-notexpired2.log", time.Now()},
				},
				time.Now().Add(-time.Hour),
				2,
			},
			[]string{"prefixA-123-expired1.log", "prefixA-123-expired2.log", "prefixA-123-expired3.log"},
		},
		{
			"Rotate 2 files based on cutoff, with absolute path",
			args{
				[]fs.FileInfo{
					&mockFile{"/path/to/prefixA-123-expired1.log", time.Now().Add(-time.Hour * 2)},
					&mockFile{"/path/to/prefixA-123-notexpired1.log", time.Now()},
					&mockFile{"/path/to/prefixA-123-expired2.log", time.Now().Add(-time.Hour * 2)},
					&mockFile{"/path/to/prefixA-123-expired3.log", time.Now().Add(-time.Hour * 2)},
					&mockFile{"/path/to/prefixA-123-notexpired2.log", time.Now()},
				},
				time.Now().Add(-time.Hour),
				2,
			},
			[]string{"/path/to/prefixA-123-expired1.log", "/path/to/prefixA-123-expired2.log", "/path/to/prefixA-123-expired3.log"},
		},
		{
			"Rotate 2 files, keep most recent",
			args{
				[]fs.FileInfo{
					&mockFile{"prefixA-123-expired1.log", time.Now().Add(-time.Hour * 5)},
					&mockFile{"prefixA-123-expired2.log", time.Now().Add(-time.Hour * 2)},
					&mockFile{"prefixA-123-expired3.log", time.Now().Add(-time.Hour * 4)},
					&mockFile{"prefixA-123-expired4.log", time.Now().Add(-time.Hour * 1)},
				},
				time.Now().Add(-time.Hour),
				2,
			},
			[]string{"prefixA-123-expired1.log", "prefixA-123-expired3.log"},
		},
		{
			"Rotate 2 files, keep most recent, multiple prefixes",
			args{
				[]fs.FileInfo{
					&mockFile{"prefixA-123-expired1.log", time.Now().Add(-time.Hour * 5)},
					&mockFile{"prefixA-123-expired2.log", time.Now().Add(-time.Hour * 2)},
					&mockFile{"prefixA-123-expired3.log", time.Now().Add(-time.Hour * 4)},
					&mockFile{"prefixB-123-expired1.log", time.Now().Add(-time.Hour * 5)},
					&mockFile{"prefixB-123-expired2.log", time.Now().Add(-time.Hour * 2)},
					&mockFile{"prefixB-123-expired3.log", time.Now().Add(-time.Hour * 4)},
				},
				time.Now().Add(-time.Hour),
				2,
			},
			[]string{"prefixA-123-expired1.log", "prefixB-123-expired1.log"},
		},
		{
			"Prefix overlap",
			args{
				[]fs.FileInfo{
					&mockFile{"prefix-123-keep1.log", time.Now()},
					&mockFile{"prefix-123-keep2.log", time.Now()},
					&mockFile{"prefix-123-expired1.log", time.Now().Add(-time.Hour * 2)},
					&mockFile{"prefix-overlap-123-keep1.log", time.Now()},
					&mockFile{"prefix-overlap-123-keep2.log", time.Now()},
					&mockFile{"prefix-overlap-123-expired1.log", time.Now().Add(-time.Hour * 2)},
				},
				time.Now().Add(-time.Hour),
				2,
			},
			[]string{"prefix-123-expired1.log", "prefix-overlap-123-expired1.log"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rotate := rotateLogs(tt.args.files, tt.args.timeCutoff, tt.args.amountCutoff)
			rotateNames := []string{}
			for _, file := range rotate {
				rotateNames = append(rotateNames, file.Name())
			}
			sort.Strings(rotateNames)
			sort.Strings(tt.want)
			assert.Equalf(t, tt.want, rotateNames, "rotateLogs(%v, %v, %v)", tt.args.files, tt.args.timeCutoff, tt.args.amountCutoff)
		})
	}
}
