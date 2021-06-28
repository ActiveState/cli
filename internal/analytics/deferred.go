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

const deferrerFileName = "deferrer"
const deferredDataFileName = "deferred_data"

func deferrerFilePath(cfg Configurable) string {
	return filepath.Join(cfg.ConfigPath(), deferrerFileName)
}

func deferredDataFilePath(cfg Configurable) string {
	return filepath.Join(cfg.ConfigPath(), deferredDataFileName)
}

func isDeferralDayAgo(cfg Configurable) bool {
	df := deferrerFilePath(cfg)
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

func runNonDeferredStateToolCommand(cfg Configurable) error {
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

func SetDeferred(cfg Configurable, da bool) {
	deferAnalytics = da
	if deferAnalytics {
		// if we have not send deferred messages for a day, run a non-deferred
		// state command in the background to flush these messages.
		if isDeferralDayAgo(cfg) {
			err := runNonDeferredStateToolCommand(cfg)
			if err != nil {
				logging.Errorf("Failed to launch non-deferred State Tool command: %v", err)
			}
		}
		return
	}
	eventWaitGroup.Add(1)
	go func() {
		defer eventWaitGroup.Done()
		if err := sendDeferred(cfg, sendEvent); err != nil {
			logging.Errorf("Could not send deferred events: %v", err)
		}
	}()
}

type Configurable interface {
	Set(string, interface{}) error
	GetString(string) string
	ConfigPath() string
}

func deferEvent(cfg Configurable, category, action, label string, dimensions map[string]string) error {
	logging.Debug("Deferring: %s, %s, %s", category, action, label)

	if !fileutils.FileExists(deferrerFilePath(cfg)) {
		if err := fileutils.Touch(deferrerFilePath(cfg)); err != nil {
			logging.Errorf("Failed to create deferrer time stamp file: %v", err)
		}
	}

	if err := saveDeferred(cfg, deferredData{category, action, label, dimensions}); err != nil {
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

func sendDeferred(cfg Configurable, sender func(string, string, string, map[string]string) error) error {
	// move deferred data file, so it is not being appended anymore
	outboxFile := filepath.Join(cfg.ConfigPath(), "deferred.outbox")
	if err := os.Rename(deferredDataFilePath(cfg), outboxFile); err != nil {
		if !os.IsNotExist(err) {
			return errs.Wrap(err, "Could not rename deferred_data file")
		}
		if err := os.Remove(deferrerFilePath(cfg)); err != nil && !os.IsNotExist(err) {
			return errs.Wrap(err, "Could not remove deferrer time stamp file")
		}
		return nil
	}
	defer os.Remove(outboxFile)

	events, err := loadDeferred(outboxFile)
	if err != nil {
		return errs.Wrap(err, "Failed to load deferred_data events")
	}
	for _, event := range events {
		if err := sender(event.Category, event.Action, event.Label, event.Dimensions); err != nil {
			return errs.Wrap(err, "Could not send deferred event")
		}
	}

	// remove deferrer time stamp file
	err = os.Remove(deferrerFilePath(cfg))
	if err != nil && !os.IsNotExist(err) {
		return errs.Wrap(err, "Could not remove deferrer time stamp file")
	}

	return nil
}

func saveDeferred(cfg Configurable, v deferredData) error {
	vj, err := json.Marshal(v)
	if err != nil {
		return errs.Wrap(err, "Failed to marshal deferred data")
	}
	f, err := os.OpenFile(deferredDataFilePath(cfg), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errs.Wrap(err, "Failed to open deferred_data file for appending")
	}
	defer f.Close()

	if _, err := f.WriteString(fmt.Sprintf("%s\n", string(vj))); err != nil {
		return errs.Wrap(err, "Failed to append deferred data")
	}
	return nil
}
