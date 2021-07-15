package buildlogfile

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/buildlog"
)

// maxConcurrency is the maximum number of artifact-build-logs that we download in parallel in order to add the information to the build log file.
const maxConcurrency = 5

type BuildLogFile struct {
	out              output.Outputer
	logFile          *os.File
	numInstalled     int
	numToBeInstalled int
	wg               *sync.WaitGroup
	ctx              context.Context
	cancel           context.CancelFunc
	logsCh           chan artifactLog
}

type artifactLog struct {
	artifactID     artifact.ArtifactID
	artifactName   string
	unsignedLogURI string
}

// verboseLogging returns true if the user provided an environment variable for it
func verboseLogging() bool {
	return os.Getenv(constants.LogBuildVerboseEnvVarName) == "true"
}

// downloadAndAddArtifactLog downloads an artifact build log and adds it to the build log file
func (bl *BuildLogFile) downloadAndAddArtifactLog(ul artifactLog, ctx context.Context) error {
	unsignedURL, err := url.Parse(ul.unsignedLogURI)
	if err != nil {
		return errs.Wrap(err, "Could not parse log URL %s", ul.unsignedLogURI)
	}
	logURL, err := model.SignS3URL(unsignedURL)
	if err != nil {
		return errs.Wrap(err, "Could not sign log url %s", unsignedURL)
	}

	// download the log and stream it line-by-line
	req, err := http.NewRequestWithContext(ctx, "GET", logURL.String(), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to create GET request for logURL.")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errs.Wrap(err, "Failed to execute HTTP GET request for logURL.")
	}

	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		// we need to unmarshal every line
		var am buildlog.ArtifactProgressMessage
		if err := json.Unmarshal(line, &am); err != nil {
			return errs.Wrap(err, "Failed to unmarshal build log line")
		}
		if err := bl.BuildArtifactProgress(ul.artifactID, ul.artifactName, am.Timestamp, am.Body.Message, am.Body.Facility, am.PipeName, am.Source); err != nil {
			return errs.Wrap(err, "Failed to write build log line")
		}
	}
	return nil
}

func New(out output.Outputer) (*BuildLogFile, error) {
	logFile, err := os.CreateTemp("", fmt.Sprintf("build-log-%s", time.Now().Format("060102030405")))
	if err != nil {
		return nil, errs.Wrap(err, "Failed to create temporary build log file")
	}

	bl := &BuildLogFile{out: out, logFile: logFile}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	bl.logsCh = make(chan artifactLog)
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ul := range bl.logsCh {
				if err := bl.downloadAndAddArtifactLog(ul, ctx); err != nil {
					logging.Error("Failed to add build log details to log file for %s: %v", ul.artifactName)
				}
			}
		}()
	}

	bl.ctx = ctx
	bl.cancel = cancel
	bl.wg = &wg

	// add info on how to activate verbose logging to first line of logfile
	if !verboseLogging() {
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
	// if verbose logging is enabled and the build was cached (ie., built in the past), we have to download the logs and add them to the file
	if verboseLogging() && isCached {
		bl.addBuildLogs(artifactID, artifactName, unsignedLogURI)
	}
	return bl.writeArtifactMessage(artifactID, artifactName, "Build completed successfully")
}

// BuildArtifactProgress writes a log status line from a specific artifact build to the logfile
func (bl *BuildLogFile) BuildArtifactProgress(artifactID artifact.ArtifactID, artifactName string, timeStamp, message, _, _, _ string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s: %s", timeStamp, message)
}

// BuildArtifactFailure writes a message that an artifact build has completed with a failure
func (bl *BuildLogFile) BuildArtifactFailure(artifactID artifact.ArtifactID, artifactName, unsignedLogURI string, errMsg string, isCached bool) error {
	// if the build is cached (ie., built in the past), we have to download its logs
	if isCached {
		bl.addBuildLogs(artifactID, artifactName, unsignedLogURI)
	}
	return bl.writeArtifactMessage(artifactID, "Build failed with error: %s", errMsg)
}

// InstallationStarted notifies the user that the process of installing the artifacts on the user's machine has started
func (bl *BuildLogFile) InstallationStarted(totalArtifacts int64) error {
	bl.numInstalled = 0
	bl.numToBeInstalled = int(totalArtifacts)
	return bl.writeMessage("== Installation started: (0/%d) ==", totalArtifacts)
}

// InstallationIncrements writes a Installation status update writing the current and total number of packages to be installed
func (bl *BuildLogFile) InstallationIncrement() error {
	bl.numInstalled++
	return bl.writeMessage("== Installation status: (%d/%d) ==", bl.numInstalled, bl.numToBeInstalled)
}

// ArtifactStepStarted writes a message that artifact update step has begun (download, unpack, install)
func (bl *BuildLogFile) ArtifactStepStarted(artifactID artifact.ArtifactID, artifactName, title string, _ int64, _ bool) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s step started", title)
}

// ArtifactStepIncrement is ignored
func (bl *BuildLogFile) ArtifactStepIncrement(_ artifact.ArtifactID, _, _ string, _ int64) error {
	return nil
}

// ArtifactStepCompleted writes a message that an artifact update step has completed successfully
func (bl *BuildLogFile) ArtifactStepCompleted(artifactID artifact.ArtifactID, artifactName, title string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s step completed SUCCESSFULLY", title)
}

// ArtifactStepFailure writes a message that an artifact update step has completed with a failure
func (bl *BuildLogFile) ArtifactStepFailure(artifactID artifact.ArtifactID, artifactName, title, errorMessage string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s step completed with FAILURE: %s", title, errorMessage)
}

// Close closes the log file after waiting for 1 second to finish up downloading all the remote log files
func (bl *BuildLogFile) Close() error {
	// wait 1 more second before we cancel the background thread
	select {
	case <-bl.ctx.Done():
	case <-time.After(time.Second):
	}
	bl.cancel()
	close(bl.logsCh)
	bl.wg.Wait()

	return bl.logFile.Close()
}

func (bl *BuildLogFile) addBuildLogs(artifactID artifact.ArtifactID, artifactName string, unsignedLogURI string) {
	bl.logsCh <- artifactLog{artifactID: artifactID, artifactName: artifactName, unsignedLogURI: unsignedLogURI}
}
