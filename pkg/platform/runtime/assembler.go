package runtime

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
)

// EnvGetter provides a function to return variables for a runtime environment
type EnvGetter interface {
	// GetEnv returns a map between environment variable names and their values
	GetEnv(inherit bool, projectDir string) (map[string]string, error)
}

// Assembler provides functionality to assemble a runtime environment for an
// installation It is usually created by an installer.Installer and defines
// which artifact tarballs to unpack where.
// Once assembled, the Assembler can be used as an EnvGetter interface to get
// the environment variables that need to be set to use the installed runtime.
type Assembler interface {
	EnvGetter
	DownloadDirectoryProvider

	// ArtifactsToDownloadAndUnpack returns the artifacts that need to be
	// downloaded and the archives that already exist and *only* need to be unpacked.
	// The second return value is a map from the archive paths to the artifact
	// instances.
	ArtifactsToDownloadAndUnpack() ([]*HeadChefArtifact, map[string]*HeadChefArtifact)

	// BuildEngine returns the build engine that this runtime has been created
	// with
	BuildEngine() BuildEngine

	// InstallerExtension is used to identify whether an artifact is one that we
	// should care about
	InstallerExtension() string

	// Unarchiver initializes and returns the unarchiver for the expected
	// artifact archive format
	Unarchiver() unarchiver.Unarchiver

	/* HOOKS */

	// PreInstall is invoked by the installer after all artifact archives are
	// downloaded, but before they are unpacked.
	PreInstall() *failures.Failure

	// PreUnpackArtifact is invoked by the installer for every artifact archive
	// before it is being unpacked.
	PreUnpackArtifact(artf *HeadChefArtifact) *failures.Failure

	// PostUnpackArtifact is invoked by the installer for every artifact archive
	// after it has been unpacked into its temporary installation directory tmpRuntimeDir
	// Here, the final relocation to InstallationDirectory() needs to take place.
	PostUnpackArtifact(artf *HeadChefArtifact, tmpRuntimeDir string, archivePath string, cb func()) *failures.Failure

	// PostInstall is called after all artifacts have been successfully installed
	PostInstall() error

	// IsInstalled returns whether the artifacts have been successfully installed already
	IsInstalled() bool
}

// BuildEngine describes the build engine that was used to build the runtime
type BuildEngine int

const (
	// UnknownEngine represents an engine unknown to the runtime.
	UnknownEngine BuildEngine = iota

	// Camel is the legacy build engine, that builds Active{Python,Perl,Tcl}
	// distributions
	Camel

	// Alternative is the new alternative build orchestration framework
	Alternative

	// Hybrid wraps Camel.
	Hybrid
)

// BuildEngineFromResponse handles a headchef build status response and returns
// the relevant engine.
func BuildEngineFromResponse(resp *headchef_models.BuildStatusResponse) BuildEngine {
	if resp == nil || resp.BuildEngine == nil {
		return UnknownEngine
	}

	switch *resp.BuildEngine {
	case headchef_models.BuildStatusResponseBuildEngineCamel:
		return Camel
	case headchef_models.BuildStatusResponseBuildEngineAlternative:
		return Alternative
	case headchef_models.BuildStatusResponseBuildEngineHybrid:
		return Hybrid
	default:
		return UnknownEngine
	}
}

func (be BuildEngine) String() string {
	switch be {
	case Camel:
		return headchef_models.BuildStatusResponseBuildEngineCamel
	case Alternative:
		return headchef_models.BuildStatusResponseBuildEngineAlternative
	case Hybrid:
		return headchef_models.BuildStatusResponseBuildEngineHybrid
	default:
		return "unknown"
	}
}
