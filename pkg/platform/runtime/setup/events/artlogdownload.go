package events

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/buildlog"
	"github.com/gammazero/workerpool"
)

// maxConcurrency is the maximum number of artifact-build-logs that we download in parallel in order to add the information to the build log file.
const maxConcurrency = 5

type ArtifactLogDownload struct {
	wp     *workerpool.WorkerPool
	events chan<- SetupEventer
	close  func()
	ctx    context.Context
}

// downloadAndArtifactLog downloads an artifact build log and adds it to the build log file
func (d *ArtifactLogDownload) downloadArtifactLog(ctx context.Context, artifactID artifact.ArtifactID, unsignedLogURI string) error {
	unsignedURL, err := url.Parse(unsignedLogURI)
	if err != nil {
		return errs.Wrap(err, "Could not parse log URL %s", unsignedLogURI)
	}
	logURL, err := model.SignS3URL(unsignedURL)
	if err != nil {
		return errs.Wrap(err, "Could not sign log url %s", unsignedURL)
	}

	// download the log and stream it line-by-line
	logging.Debug("downloading logURI: %s", logURL.String())
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
		d.events <- newArtifactBuildProgressEvent(artifactID, am.Timestamp, am.Body.Message, am.Body.Facility, am.PipeName, am.Source)
	}
	return nil
}

func NewArtifactLogDownload(events chan<- SetupEventer) *ArtifactLogDownload {
	ctx, cancel := context.WithCancel(context.Background())

	wp := workerpool.New(maxConcurrency)

	close := func() {
		logging.Debug("closing artifact log download instance")
		done := make(chan struct{})
		// cancel context after worker pool is done processing all events
		go func() {
			wp.StopWait()
			cancel()
			close(done)
		}()

		// check if we are done OR if shut-down takes more than 5 minutes, cancel all HTTP requests
		select {
		case <-done:
		case <-time.After(time.Second * 5):
			logging.Error("Failed to process all artifact log downloads.")
			cancel()
			<-done
		}
	}
	return &ArtifactLogDownload{events: events, ctx: ctx, wp: wp, close: close}
}

func (d *ArtifactLogDownload) RequestArtifactLog(artifactID artifact.ArtifactID, unsignedLogURI string) {
	logging.Debug("submitting artifact log message for artifact %s to be added to log", artifactID)
	d.wp.Submit(func() {
		if err := d.downloadArtifactLog(d.ctx, artifactID, unsignedLogURI); err != nil {
			logging.Error("Failed to add build log details to log file for %s: %v", artifactID, errs.JoinMessage(err))
		}
	})
}

func (d *ArtifactLogDownload) Close() {
	d.close()
}
