package exeutils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FilterExesOnPATH(t *testing.T) {
	fileExistsIfEquals := func(equals string) func(string) bool {
		return func(f string) bool { return f==equals }
	}
	fileExistsTrue := func(f string) bool { return true }
	fileExistsFalse := func(f string) bool { return false }
	filterTrue := func(string) bool { return true}
	filterFalse := func(string) bool { return false}

	PATH := strings.Join([]string{"test2", "test"}, string(os.PathListSeparator))

	assert.Equal(t,
		filepath.Join("test", "state"),
		findExe("state", PATH, []string{}, fileExistsIfEquals(filepath.Join("test", "state")), filterTrue),
	)
	assert.Equal(t,
		filepath.Join("test", "state.exe"),
		findExe("state", PATH, []string{".exe"}, fileExistsIfEquals(filepath.Join("test", "state.exe")), filterTrue),
	)
	assert.Equal(t,
		filepath.Join("test", "state.exe"),
		findExe("state.exe", PATH, []string{}, fileExistsIfEquals(filepath.Join("test", "state.exe")), filterTrue),
	)
	assert.Equal(t,
		filepath.Join("test", "state"),
		findExe("state", PATH, []string{}, fileExistsIfEquals(filepath.Join("test", "state")), filterTrue),
	)
	assert.Equal(t, "", findExe("state", PATH, []string{}, fileExistsTrue, filterFalse))
	assert.Equal(t, "", findExe("non-existent", PATH, []string{}, fileExistsFalse, filterTrue))
	assert.Equal(t, "", findExe("state", PATH, []string{}, fileExistsFalse, filterTrue))
}
