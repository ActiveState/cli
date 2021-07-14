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

func verboseLogging() bool {
	return os.Getenv(constants.LogBuildVerboseEnvVarName) == "true"
}

func (bl *BuildLogFile) handleLog(ul artifactLog, ctx context.Context) error {
	// download log
	unsignedURL, err := url.Parse(ul.unsignedLogURI)
	if err != nil {
		return errs.Wrap(err, "Could not parse log URL %s", ul.unsignedLogURI)
	}
	logURL, err := model.SignS3URL(unsignedURL)
	if err != nil {
		return errs.Wrap(err, "Could not sign log url %s", unsignedURL)
	}

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

	bl := &BuildLogFile{logFile: logFile}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	bl.logsCh = make(chan artifactLog)
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ul := range bl.logsCh {
				if err := bl.handleLog(ul, ctx); err != nil {
					logging.Error("Failed to add build log details to log file for %s: %v", ul.artifactName)
				}
			}
		}()
	}

	bl.ctx = ctx
	bl.cancel = cancel
	bl.wg = &wg

	if !verboseLogging() {
		bl.writeMessage("To increase the verbosity of these log files use the %s=true environment variable", constants.LogBuildVerboseEnvVarName)
	}

	return bl, nil
}

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

func (bl *BuildLogFile) BuildStarted(totalArtifacts int64) error {
	bl.out.Print(locale.Tl("view_build_logfile_info", "View the Build Log to follow the build progress in detail: [ACTIONABLE]{{.V0}}[/RESET]", bl.logFile.Name()))
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

func (bl *BuildLogFile) BuildArtifactCompleted(artifactID artifact.ArtifactID, artifactName string, unsignedLogURI string, isCached bool) error {
	if verboseLogging() && isCached {
		bl.addBuildLogs(artifactID, artifactName, unsignedLogURI)
	}
	return bl.writeArtifactMessage(artifactID, artifactName, "Build completed successfully")
}

func (bl *BuildLogFile) BuildArtifactProgress(artifactID artifact.ArtifactID, artifactName string, timeStamp, message, _, _, _ string) error {
	return bl.writeArtifactMessage(artifactID, artifactName, "%s: %s", timeStamp, message)
}

func (bl *BuildLogFile) BuildArtifactFailure(artifactID artifact.ArtifactID, artifactName, unsignedLogURI string, errMsg string, isCached bool) error {
	if verboseLogging() || isCached {
		bl.addBuildLogs(artifactID, artifactName, unsignedLogURI)
	}
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
