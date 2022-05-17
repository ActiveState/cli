package sysinfo

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/performance"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPerformance tests the speed at which critical sysinfo methods execute
// These timing are critical because we query this information at each invocation of State Tool
func TestPerformance(t *testing.T) {
	oldCache := sysinfoCache
	defer func() { sysinfoCache = oldCache }()
	sysinfoCache = cache.New(0, 0)

	t.Run("OS", func(t *testing.T) {
		maxDuration := time.Microsecond
		err := performance.TimeIt(t, func() { OS() }, 10, maxDuration)
		assert.NoError(t, err)
	})

	t.Run("OSVersion", func(t *testing.T) {
		maxDuration := 10 * time.Millisecond
		err := performance.TimeIt(t, func() {
			_, err := OSVersion()
			assert.NoError(t, err)
		}, 10, maxDuration)
		assert.NoError(t, err)
	})
}

func TestGetDarwinProductVersionFromFS(t *testing.T) {
	productVersion, err := getDarwinProductVersionFromFS()
	require.NoError(t, err)
	assert.NotEmpty(t, productVersion)
}

func TestOSVersionInfoCached(t *testing.T) {
	osVersionInfo1, _ := OSVersion()
	osVersionInfo2, _ := OSVersion()
	assert.True(t, osVersionInfo1 != nil, "OSVersion should not be nil")
	assert.True(t, osVersionInfo1 == osVersionInfo2, "Pointers should be equal")
}

func TestLibcInfoCached(t *testing.T) {
	libcInfo1, _ := Libc()
	libcInfo2, _ := Libc()
	assert.True(t, libcInfo1 != nil, "Libc should not be nil")
	assert.True(t, libcInfo1 == libcInfo2, "Pointers should be equal")
}

func TestCompilersCached(t *testing.T) {
	compilers1, _ := Compilers()
	compilers2, _ := Compilers()
	assert.True(t, compilers1 != nil, "Compilers should not be nil")
	for i := range compilers1 {
		assert.True(t, compilers1[i] == compilers2[i], "Pointers should be equal")
	}
}
