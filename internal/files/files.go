package files

import (
	"path/filepath"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/qor/assetfs"
)

func init() {
	root, err := environment.GetRootPath()
	if err != nil {
		panic(err)
	}

	assetfs.SetAssetFS(AssetFS)

	AssetFS.RegisterPath(filepath.Join(root, "assets"))

	AssetFS.Compile()
}
