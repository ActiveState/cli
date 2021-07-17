package buildlogfile

import (
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type BuildLogFile struct {
	out     output.Outputer
	logFile *os.File
}

// verboseLogging is true if the user provided an environment variable for it
var verboseLogging = os.Getenv(constants.LogBuildVerboseEnvVarName) == "true"

func New(out output.Outputer) (*BuildLogFile, error) {
	logFile, err := os.CreateTemp("", fmt.Sprintf("build-log-%s", time.Now().Format("060102030405")))
	if err != nil {
		return nil, errs.Wrap(err, "Failed to create temporary build log file")
	}

	bl := &BuildLogFile{out: out, logFile: logFile}

	// add info on how to activate verbose logging to first line of logfile
	if !verboseLogging {
		bl.writeMessage("To increase the verbosity of these log files use the %s=true environment variable", constants.LogBuildVerboseEnvVarName)
	}

	return bl, nil
}

// Path returns the absolute path to the log file
func (bl *BuildLogFile) Path() string {
	return bl.logFile.Name()
}

func (bl *BuildLogFile) writeMessage(msg string, values ...interface{}) error {
	_, err := bl.logFile.WriteString(fmt.Sprintf(msg+"\n", values...))
	if err != nil {
		return errs.Wrap(err, "Failed to write message to log file.")
	}
	return nil
}

func (bl *BuildLogFile) writeArtifactMessage(artifactID artifact.ArtifactID, artifactName string, msg string, values ...interface{}) error {
	_, err := bl.logFile.WriteString(fmt.Sprintf("[%s (%s)] ", artifactName, artifactID.String()) + fmt.Sprintf(msg+"\n", values...))
	if err != nil {
		return errs.Wrap(err, "Failed to write message to log file.")
	}
	return nil
}

// BuildStarted writes a message that the build has started remotely
func (bl *BuildLogFile) BuildStarted(totalArtifacts int64) error {
	// also print out a message about the log file location
	bl.out.Print(locale.Tl("view_build_logfile_info", "View the Build Log to follow the build progress in detail: [ACTIONABLE]{{.V0}}[/RESET]", bl.logFile.Name()))
	return bl.writeMessage("== Scheduled building of %d artifacts ==", totalArtifacts)
}

// BuildCompleted writes a message that the build has completed
func (bl *BuildLogFile) BuildCompleted(withFailures bool) error {
	outcome := "SUCCESSFULLY"
	if withFailures {
		outcome = "with FAILURES"
	}
	return bl.writeMessage("== Build completed %s. ==", outcome)
}

// StillBuilding is called if there was no other build message for 15 seconds
func (bl *BuildLogFile) StillBuilding(numCompleted, numTotal int) error {
	return bl.writeMessage("== Still building [%d/%d] ==", numCompleted, numTotal)
}

// BuildArtifactStarted writes a message that a new artifact has started building
func (bl *BuildLogFile) BuildArtifactStarted(artifactID artifact.ArtifactID, artifactName string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "Build started")
}

// BuildArtifactCompleted writes a message that an artifact build has completed successfully
func (bl *BuildLogFile) BuildArtifactCompleted(artifactID artifact.ArtifactID, artifactName string, unsignedLogURI string, isCached bool) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "Build completed successfully")
}

// BuildArtifactProgress writes a log status line from a specific artifact build to the logfile
func (bl *BuildLogFile) BuildArtifactProgress(artifactID artifact.ArtifactID, artifactName string, timeStamp, message, _, _, _ string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s: %s", timeStamp, message)
}

// BuildArtifactFailure writes a message that an artifact build has completed with a failure
func (bl *BuildLogFile) BuildArtifactFailure(artifactID artifact.ArtifactID, artifactName, unsignedLogURI string, errMsg string, isCached bool) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "Build failed with error: %s", errMsg)
}

// InstallationStarted notifies the user that the process of installing the artifacts on the user's machine has started
func (bl *BuildLogFile) InstallationStarted(totalArtifacts int64) error {
	return bl.writeMessage("== Installation started: (0/%d) ==", totalArtifacts)
}

// InstallationStatusUpdate writes an Installation status update with the current and total number of packages to be installed
func (bl *BuildLogFile) InstallationStatusUpdate(current, total int64) error {
	return bl.writeMessage("== Installation status: (%d/%d) ==", current, total)
}

// ArtifactStepStarted writes a message that artifact update step has begun (download, unpack, install)
func (bl *BuildLogFile) ArtifactStepStarted(artifactID artifact.ArtifactID, artifactName, title string, _ int64, _ bool) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s started", title)
}

// ArtifactStepIncrement is ignored
func (bl *BuildLogFile) ArtifactStepIncrement(_ artifact.ArtifactID, _, _ string, _ int64) error {
	return nil
}

// ArtifactStepCompleted writes a message that an artifact update step has completed successfully
func (bl *BuildLogFile) ArtifactStepCompleted(artifactID artifact.ArtifactID, artifactName, title string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s completed SUCCESSFULLY", title)
}

// ArtifactStepFailure writes a message that an artifact update step has completed with a failure
func (bl *BuildLogFile) ArtifactStepFailure(artifactID artifact.ArtifactID, artifactName, title, errorMessage string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s completed with FAILURE: %s", title, errorMessage)
}

func (bl *BuildLogFile) Close() error {
	return nil
}
