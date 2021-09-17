package deferred

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
)

type DeferredData struct {
	Category    string
	Action      string
	Label       string
	ProjectName string
	Output      string
}

const deferrerFileName = "deferrer_data"

func DeferrerFilePath() string {
	appDataPath, err := storage.AppDataPath()
	if err != nil {
		logging.Error("Failed to get AppDataPath: %s", errs.JoinMessage(err))
	}
	return filepath.Join(appDataPath, deferrerFileName)
}

func DeferEvent(category, action, label, projectName, output string) error {
	logging.Debug("Deferring: %s, %s, %s", category, action, label, projectName, output)

	if err := saveDeferred(DeferredData{category, action, label, projectName, output}); err != nil {
		return errs.Wrap(err, "Could not save event on defer")
	}
	return nil
}

func LoadEvents() ([]DeferredData, error) {
	appDataPath, err := storage.AppDataPath()
	if err != nil {
		return nil, errs.Wrap(err, "Could not retrieve AppDataPath")
	}

	// move deferred data file, so it is not being appended anymore
	outboxFile := filepath.Join(appDataPath, fmt.Sprintf("deferred.%d-%d", os.Getpid(), time.Now().Unix()))
	if err := os.Rename(DeferrerFilePath(), outboxFile); err != nil {
		if !os.IsNotExist(err) {
			return nil, errs.Wrap(err, "Could not rename deferred_data file")
		}
		return nil, nil // No deferred data to send
	}
	defer os.Remove(outboxFile)

	events, err := loadDeferred(outboxFile)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to load deferred events")
	}

	return events, nil
}

func saveDeferred(v DeferredData) error {
	vj, err := json.Marshal(v)
	if err != nil {
		return errs.Wrap(err, "Failed to marshal deferred data")
	}
	path := DeferrerFilePath()
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
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

	return nil
}

func loadDeferred(deferredDataPath string) ([]DeferredData, error) {
	b, err := os.ReadFile(deferredDataPath)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to read deferred_data")
	}
	lines := strings.Split(string(b), "\n")
	var events []DeferredData
	var unmarshalErrorReported bool
	for _, line := range lines {
		var event DeferredData
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
