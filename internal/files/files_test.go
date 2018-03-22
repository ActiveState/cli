package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/stretchr/testify/assert"
)

func TestShellsAssets(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Root path detected")
	foundAnAsset := false
	filepath.Walk(filepath.Join(root, "assets", "shells"), func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			asset, err := AssetFS.Asset(filepath.Join("shells", filepath.Base(path)))
			assert.NoError(t, err, "Retrieved the shell asset")
			assert.NotNil(t, asset, "Shell asset is non-nil")
			foundAnAsset = true
		}
		return nil
	})
	assert.True(t, foundAnAsset, "Shell assets were found")
}
