package analytics

import (
	"sync"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type CustomDimensions struct {
	version       string
	branchName    string
	userID        string
	output        string
	osName        string
	osVersion     string
	installSource string
	machineID     string
	projectName   string
	mu            sync.Mutex
}

func (d *CustomDimensions) SetOutput(output string) {
	logging.Debug("setting output field to: %s", output)

	d.mu.Lock()
	defer d.mu.Unlock()

	d.output = output
}

func (d *CustomDimensions) toMap() map[string]string {
	d.mu.Lock()
	defer d.mu.Unlock()

	pj := projectfile.GetPersisted()
	d.projectName = ""
	if pj != nil {
		d.projectName = pj.Owner() + "/" + pj.Name()
	}

	return map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprected)"
		"2":  d.version,
		"3":  d.branchName,
		"4":  d.userID,
		"5":  d.output,
		"6":  d.osName,
		"7":  d.osVersion,
		"8":  d.installSource,
		"9":  d.machineID,
		"10": d.projectName,
	}
}
