package installation

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallRoot(t *testing.T) {
	tempdirWithInstall := fileutils.TempDirUnsafe()
	tempdirWithOutInstall := fileutils.TempDirUnsafe()
	err := fileutils.Touch(filepath.Join(tempdirWithInstall, InstallDirMarker))
	require.NoError(t, err)

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			"Root resolves to root",
			tempdirWithInstall,
			tempdirWithInstall,
			false,
		},
		{
			"Subdir resolves to root",
			filepath.Join(tempdirWithInstall, "subdir"),
			tempdirWithInstall,
			false,
		},
		{
			"Not an install will error",
			filepath.Join(tempdirWithOutInstall, "subdir"),
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InstallRoot(tt.path)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equalf(t, tt.want, got, "InstallRoot(%v)", tt.path)
		})
	}
}

func TestBinPathFromInstallPath(t *testing.T) {
	tempdirWithInstall := fileutils.TempDirUnsafe()
	tempdirWithOutInstall := fileutils.TempDirUnsafe()
	err := fileutils.Touch(filepath.Join(tempdirWithInstall, InstallDirMarker))
	require.NoError(t, err)

	tests := []struct {
		name        string
		installPath string
		want        string
		wantErr     bool
	}{
		{
			"Returns root bin dir",
			filepath.Join(tempdirWithInstall, "subdir"),
			filepath.Join(tempdirWithInstall, BinDirName),
			false,
		},
		{
			"Errors out if given path does not have an install",
			filepath.Join(tempdirWithOutInstall, "subdir"),
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BinPathFromInstallPath(tt.installPath)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equalf(t, tt.want, got, "BinPathFromInstallPath(%v)", tt.installPath)
		})
	}
}

func TestInstallPathFromReference(t *testing.T) {
	installPathSuffix := filepath.Join(".ActiveState", "StateTool", "release")
	if runtime.GOOS == "windows" {
		installPathSuffix = filepath.Join("AppData", "Local", "ActiveState", "StateTool", "release")
	}

	home := fileutils.TempDirUnsafe()
	installDir := filepath.Join(home, installPathSuffix)
	err := fileutils.Mkdir(home, installPathSuffix)
	require.NoError(t, err)

	err = fileutils.Touch(filepath.Join(installDir, InstallDirMarker))
	require.NoError(t, err)

	binDir := filepath.Join(installDir, "bin")
	err = fileutils.Mkdir(binDir)
	require.NoError(t, err)

	err = fileutils.Touch(filepath.Join(binDir, constants.StateCmd+osutils.ExeExtension))
	require.NoError(t, err)

	t.Setenv(constants.HomeEnvVarName, home)
	_, err = InstallPathFromReference(filepath.Join(installDir, "bin"))
	require.NoError(t, err)
}
