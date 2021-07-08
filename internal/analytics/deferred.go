package analytics

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
)

var deferAnalytics bool

type deferredData struct {
	Category   string
	Action     string
	Label      string
	Dimensions map[string]string
}

const deferrerFileName = "deferrer_data"
const deferrerTimestampFileName = "deferrer"

func deferrerFilePath() string {
	appDataPath, err := storage.AppDataPath()
	if err != nil {
		logging.Error("Failed to get AppDataPath: %s", errs.JoinMessage(err))
	}
	return filepath.Join(appDataPath, deferrerFileName)
}

func deferrerTimeFilePath() string {
	appDataPath, err := storage.AppDataPath()
	if err != nil {
		logging.Error("Failed to get AppDataPath: %s", errs.JoinMessage(err))
	}
	return filepath.Join(appDataPath, deferrerTimestampFileName)
}

func isDeferralDayAgo() bool {
	df := deferrerTimeFilePath()
	stat, err := os.Stat(df)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		logging.Error("Could not stat deferrer file: %s, error: %v", df, err)
		return false
	}

	diff := time.Now().Sub(stat.ModTime())
	return diff > 24*time.Hour
}

func runNonDeferredStateToolCommand() error {
	exe, err := os.Executable()
	if err != nil {
		logging.Errorf("Could not determine State Tool executable: %v", err)
		exe = "state"
	}
	cmd := exec.Command(exe, "--version")
	cmd.SysProcAttr = osutils.SysProcAttrForNewProcessGroup()
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	err = cmd.Start()
	if err != nil {
		return errs.Wrap(err, "Failed to run %s --version in background", exe)
	}
	err = cmd.Process.Release()
	if err != nil {
		return errs.Wrap(err, "Failed to release process resources for background process")
	}

	return nil
}

func SetDeferred(da bool) {
	deferAnalytics = da
	if deferAnalytics {
		// if we have not send deferred messages for a day, run a non-deferred
		// state command in the background to flush these messages.
		if isDeferralDayAgo() {
			err := runNonDeferredStateToolCommand()
			if err != nil {
				logging.Errorf("Failed to launch non-deferred State Tool command: %v", err)
			}
		}
		return
	}

	// If we are not in a deferred state then we flush the deferred events that have been queued up
	eventWaitGroup.Add(1)
	go func() {
		defer eventWaitGroup.Done()
		if err := sendDeferred(sendEvent); err != nil {
			logging.Errorf("Could not send deferred events: %v", err)
		}
	}()
}

type Configurable interface {
	Set(string, interface{}) error
	GetString(string) string
	ConfigPath() string
}

func deferEvent(category, action, label string, dimensions map[string]string) error {
	logging.Debug("Deferring: %s, %s, %s", category, action, label)

	if err := saveDeferred(deferredData{category, action, label, dimensions}); err != nil {
		return errs.Wrap(err, "Could not save event on defer")
	}
	return nil
}

func loadDeferred(deferredDataPath string) ([]deferredData, error) {
	b, err := os.ReadFile(deferredDataPath)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to read deferred_data")
	}
	lines := strings.Split(string(b), "\n")
	var events []deferredData
	var unmarshalErrorReported bool
	for _, line := range lines {
		var event deferredData
		if strings.TrimSpace(line) == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			if !unmarshalErrorReported {
				logging.Error("Failed to unmarshal line in deferred_data file: %v", err)
				unmarshalErrorReported = true
			}
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

func sendDeferred(sender func(string, string, string, map[string]string) error) error {
	appDataPath, err := storage.AppDataPath()
	if err != nil {
		return errs.Wrap(err, "Could not retrieve AppDataPath")
	}

	tsPath := deferrerTimeFilePath()
	if fileutils.FileExists(tsPath) {
		if err := os.Remove(tsPath); err != nil {
			return errs.Wrap(err, "Could not remove timestamp file: %s", tsPath)
		}
	}

	// move deferred data file, so it is not being appended anymore
	outboxFile := filepath.Join(appDataPath, fmt.Sprintf("deferred.%d-%d", os.Getpid(), time.Now().Unix()))
	if err := os.Rename(deferrerFilePath(), outboxFile); err != nil {
		if !os.IsNotExist(err) {
			return errs.Wrap(err, "Could not rename deferred_data file")
		}
		return nil // No deferred data to send
	}
	defer os.Remove(outboxFile)

	events, err := loadDeferred(outboxFile)
	if err != nil {
		return errs.Wrap(err, "Failed to load deferred events")
	}
	for _, event := range events {
		if err := sender(event.Category, event.Action, event.Label, event.Dimensions); err != nil {
			return errs.Wrap(err, "Could not send deferred event")
		}
	}

	return nil
}

func saveDeferred(v deferredData) error {
	vj, err := json.Marshal(v)
	if err != nil {
		return errs.Wrap(err, "Failed to marshal deferred data")
	}
	path := deferrerFilePath()
	if err := os.MkdirAll(filepath.Dir(path), os.ModeDir); err != nil {
		return errs.Wrap(err, "Failed to create deferred file dir")
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errs.Wrap(err, "Failed to open deferred_data file for appending")
	}
	defer f.Close()

	if _, err := f.WriteString(fmt.Sprintf("%s\n", string(vj))); err != nil {
		return errs.Wrap(err, "Failed to append deferred data")
	}

	tsPath := deferrerTimeFilePath()
	if !fileutils.FileExists(tsPath) {
		if err := fileutils.Touch(tsPath); err != nil {
			return errs.Wrap(err, "Could not touch deferred timestamp file")
		}
	}

	return nil
}
