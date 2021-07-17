package events

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/buildlog"
)

// maxConcurrency is the maximum number of artifact-build-logs that we download in parallel in order to add the information to the build log file.
const maxConcurrency = 5

type artifactLog struct {
	artifactID     artifact.ArtifactID
	unsignedLogURI string
}

type ArtifactLogDownload struct {
	logfiles chan artifactLog
	events   chan<- SetupEventer
	close    func()
}

// downloadAndArtifactLog downloads an artifact build log and adds it to the build log file
func (d *ArtifactLogDownload) downloadArtifactLog(ul artifactLog, ctx context.Context) error {
	unsignedURL, err := url.Parse(ul.unsignedLogURI)
	if err != nil {
		return errs.Wrap(err, "Could not parse log URL %s", ul.unsignedLogURI)
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
		d.events <- newArtifactBuildProgressEvent(ul.artifactID, am.Timestamp, am.Body.Message, am.Body.Facility, am.PipeName, am.Source)
	}
	return nil
}

func NewArtifactLogDownload(events chan<- SetupEventer) *ArtifactLogDownload {
	d := &ArtifactLogDownload{events: events}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	logfiles := make(chan artifactLog)
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ul := range logfiles {
				if err := d.downloadArtifactLog(ul, ctx); err != nil {
					logging.Error("Failed to add build log details to log file for %s: %v", ul.artifactID, errs.JoinMessage(err))
				}
			}
		}()
	}

	// cancel context after ALL wait groups return -> this indicates "done"-ness
	go func() {
		defer cancel()
		wg.Wait()
	}()

	d.logfiles = logfiles
	d.close = func() {
		// close the input channel which should drain the go routines we started
		close(logfiles)
		// check if we are done (in which case the context is done)
		select {
		case <-ctx.Done():
		case <-time.After(time.Second * 5): // If it takes more than 5 seconds, cancel the context manually (which cancels current downloads)
			cancel()
			<-ctx.Done()
		}
	}
	return d
}

func (d *ArtifactLogDownload) RequestArtifactLog(artifactID artifact.ArtifactID, unsignedLogURI string) error {
	select {
	case d.logfiles <- artifactLog{artifactID, unsignedLogURI}:
	case <-time.After(1 * time.Second):
		return errs.New("Failed to request artifact logs due to timeout.")
	}
	return nil
}

func (d *ArtifactLogDownload) Close() {
	d.close()
}
