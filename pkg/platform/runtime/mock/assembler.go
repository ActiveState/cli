package mock

import (
	testifyMock "github.com/stretchr/testify/mock"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

var _ runtime.Assembler = &Assembler{}

type Assembler struct {
	testifyMock.Mock
}

func (a *Assembler) DownloadDirectory(artf *runtime.HeadChefArtifact) (string, *failures.Failure) {
	args := a.Called(artf)
	return args.String(0), args.Get(1).(*failures.Failure)
}
func (a *Assembler) GetEnv(inherit bool, projectDir string) (map[string]string, *failures.Failure) {
	args := a.Called()
	return args.Get(0).(map[string]string), args.Get(1).(*failures.Failure)
}

func (a *Assembler) ArtifactsToDownloadAndUnpack() ([]*runtime.HeadChefArtifact, map[string]*runtime.HeadChefArtifact) {
	args := a.Called()
	return args.Get(0).([]*runtime.HeadChefArtifact), args.Get(1).(map[string]*runtime.HeadChefArtifact)
}

func (a *Assembler) InstallationDirectory(artf *runtime.HeadChefArtifact) string {
	args := a.Called(artf)
	return args.String(0)
}

func (a *Assembler) InstallDirs() []string {
	args := a.Called()
	return args.Get(0).([]string)
}

func (a *Assembler) PreInstall() *failures.Failure {
	args := a.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*failures.Failure)
}

func (a *Assembler) PreUnpackArtifact(artf *runtime.HeadChefArtifact) *failures.Failure {
	args := a.Called(artf)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*failures.Failure)
}

func (a *Assembler) PostUnpackArtifact(artf *runtime.HeadChefArtifact, tmpRuntimeDir string, archivePath string, cb func()) *failures.Failure {
	args := a.Called(artf, tmpRuntimeDir, archivePath, cb)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*failures.Failure)
}

func (a *Assembler) InstallerExtension() string {
	return ".tar.gz"
}

func (a *Assembler) Unarchiver() unarchiver.Unarchiver {
	args := a.Called()
	return args.Get(0).(unarchiver.Unarchiver)
}

func (a *Assembler) BuildEngine() runtime.BuildEngine {
	args := a.Called()
	return args.Get(0).(runtime.BuildEngine)
}
