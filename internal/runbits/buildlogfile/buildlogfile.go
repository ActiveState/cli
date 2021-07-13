package buildlogfile

import (
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type BuildLogFile struct {
	logFile          *os.File
	numInstalled     int
	numToBeInstalled int
	// artifacts artifact.ArtifactRecipeMap
	/*
		wg        *sync.WaitGroup
		httpCtx   context.Context
		cancel    context.CancelFunc
	*/
}

func New() (*BuildLogFile, error) {
	logFile, err := os.CreateTemp("", "build-log*")
	if err != nil {
		return nil, errs.Wrap(err, "Failed to create temporary build log file")
	}

	return &BuildLogFile{logFile: logFile}, nil
}

func (bl *BuildLogFile) Path() string {
	return bl.logFile.Name()
}

func (bl *BuildLogFile) writeMessage(msg string, values ...interface{}) error {
	_, err := bl.logFile.WriteString(fmt.Sprintf(msg+"\n", values))
	if err != nil {
		return errs.Wrap(err, "Failed to write message to log file.")
	}
	return nil
}

func (bl *BuildLogFile) writeArtifactMessage(artifactID artifact.ArtifactID, artifactName string, msg string, values ...interface{}) error {
	_, err := bl.logFile.WriteString(fmt.Sprintf("[%s (%s)]", artifactName, artifactID.String()) + fmt.Sprintf(msg+"\n", values))
	if err != nil {
		return errs.Wrap(err, "Failed to write message to log file.")
	}
	return nil
}

func (bl *BuildLogFile) BuildStarted(totalArtifacts int64) error {
	return bl.writeMessage("== Scheduled building of %d artifacts ==", totalArtifacts)
}

func (bl *BuildLogFile) BuildCompleted(withFailures bool) error {
	outcome := "SUCCESSFULLY"
	if withFailures {
		outcome = "with FAILURES"
	}
	return bl.writeMessage("== Build completed %s. ==", outcome)
}

func (bl *BuildLogFile) BuildArtifactStarted(artifactID artifact.ArtifactID, artifactName string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "Build started")
}

func (bl *BuildLogFile) BuildArtifactCompleted(artifactID artifact.ArtifactID, artifactName string, logURI string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "Build completed successfully")
}

func (bl *BuildLogFile) LogLine(artifactID artifact.ArtifactID, artifactName string, timeStamp time.Time, msg string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, fmt.Sprintf("%s: %s", timeStamp.String(), msg))
}

func (bl *BuildLogFile) BuildArtifactFailure(artifactID artifact.ArtifactID, artifactName, unsignedLogURI string, errMsg string) error {
	return bl.writeArtifactMessage(artifactID, "Build failed with error: %s", errMsg)
}

func (bl *BuildLogFile) InstallationStarted(totalArtifacts int64) error {
	bl.numInstalled = 0
	bl.numToBeInstalled = int(totalArtifacts)
	return bl.writeMessage("== Installation started: (0/%d) ==", totalArtifacts)
}

func (bl *BuildLogFile) InstallationIncrement() error {
	bl.numInstalled++
	return bl.writeMessage("== Installation status: (%d/%d) ==", bl.numInstalled, bl.numToBeInstalled)
}

func (bl *BuildLogFile) ArtifactStepStarted(artifactID artifact.ArtifactID, artifactName, title string, _ int64, _ bool) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s step started", title)
}

func (bl *BuildLogFile) ArtifactStepIncrement(_ artifact.ArtifactID, _, _ string, _ int64) error {
	return nil
}

func (bl *BuildLogFile) ArtifactStepCompleted(artifactID artifact.ArtifactID, artifactName, title string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s step completed SUCCESSFULLY", title)
}

func (bl *BuildLogFile) ArtifactStepFailure(artifactID artifact.ArtifactID, artifactName, title, errorMessage string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s step completed with FAILURE: %s", title, errorMessage)
}

func (bl *BuildLogFile) Close() error {
	return bl.logFile.Close()
}
