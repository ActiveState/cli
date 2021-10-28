// +build !windows

package exeutils

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FilterExesOnPATH(t *testing.T) {
	fileExistsTrue := func(f string) bool { return true }
	fileExistsFalse := func(f string) bool { return false }
	filterTrue := func(string) bool { return true}
	filterFalse := func(string) bool { return false}

	assert.Equal(t, filepath.Join("test", "state"), findExe("state", "test", fileExistsTrue, filterTrue))
	assert.Equal(t, "", findExe("state", "test", fileExistsTrue, filterFalse))
	assert.Equal(t, "", findExe("non-existent", "test", fileExistsFalse, filterTrue))
	assert.Equal(t, "", findExe("state", "test", fileExistsFalse, filterTrue))
}
