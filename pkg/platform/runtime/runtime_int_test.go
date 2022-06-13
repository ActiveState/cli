package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/analytics/client/blackhole"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/runbits/buildlogfile"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/platform/runtime/testhelper"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfflineInstaller(t *testing.T) {
	// Each artifact of the form UUID.tar.gz has the following structure:
	// - runtime.json (empty)
	// - tmp (directory)
	//   - [number] (file)
	// The numbered file is the key to the following maps.
	testArtifacts := map[string]strfmt.UUID{
		"1":  "74D554B3-6B0F-434B-AFE2-9F2F0B5F32BA",
		"2":  "87ADD1B0-169D-4C01-8179-191BB9910799",
		"3":  "5D8D933F-09FA-45A3-81FF-E6F33E91C9ED",
		"4":  "992B8488-C61D-433C-ADF2-D76EBD8DAE59",
		"5":  "2C36A315-59ED-471B-8629-2663ECC95476",
		"6":  "57E8EAF4-F7EE-4BEF-B437-D9F0A967BA52",
		"7":  "E299F10C-7B5D-4B25-B821-90E30193A916",
		"8":  "F95C0ECE-9F69-4998-B83F-CE530BACD468",
		"9":  "CAC9708D-FAA6-4295-B640-B8AA41A8AABC",
		"10": "009D20C9-0E38-44E8-A095-7B6FEF01D7DA",
	}
	const artifactsPerArtifact = 2 // files/artifacts per artifact.tar.gz

	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	owner := "testOwner"
	name := "testName"
	commitID := strfmt.UUID("000000000-0000-0000-0000-000000000000")
	artifactsDir := osutil.GetTestDataDir()
	offlineTarget := target.NewInstallationTarget(owner, name, commitID, dir, artifactsDir)

	analytics := blackhole.New()
	mockProgress := &testhelper.MockProgressOutput{}
	logfile, err := buildlogfile.New(outputhelper.NewCatcher())
	require.NoError(t, err)
	eventHandler := events.NewRuntimeEventHandler(mockProgress, nil, logfile)

	rt, err := New(offlineTarget, analytics, nil)
	require.Error(t, err)
	assert.True(t, IsNeedsUpdateError(err), "runtime should require an update")
	err = rt.Update(nil, eventHandler)
	require.NoError(t, err)

	assert.False(t, mockProgress.BuildStartedCalled)
	assert.False(t, mockProgress.BuildCompletedCalled)
	assert.Equal(t, int64(0), mockProgress.BuildTotal)
	assert.Equal(t, 0, mockProgress.BuildCurrent)
	assert.Equal(t, 1, mockProgress.InstallationStartedCalled)
	assert.Equal(t, int64(len(testArtifacts)), mockProgress.InstallationTotal)
	assert.Equal(t, len(testArtifacts)*artifactsPerArtifact, mockProgress.ArtifactStartedCalled)
	assert.Equal(t, 2*len(testArtifacts)*artifactsPerArtifact, mockProgress.ArtifactIncrementCalled) // start and stop each have one count
	assert.Equal(t, len(testArtifacts)*artifactsPerArtifact, mockProgress.ArtifactCompletedCalled)
	assert.Equal(t, 0, mockProgress.ArtifactFailureCalled)

	for filename, _ := range testArtifacts {
		filename := filepath.Join(dir, "tmp", filename) // each file is in a "tmp" dir in the archive
		assert.True(t, fileutils.FileExists(filename), "file '%s' was not extracted from its artifact", filename)
	}
}
