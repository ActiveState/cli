package service

import "sync"

type CustomDimensions struct {
	version          string
	branchName       string
	userIDLock       sync.Mutex
	userID           string
	osName           string
	osVersion        string
	installSource    string
	machineID        string
	uniqID           string
	sessionToken     string
	updateTag        string
	projectNameSpace string
	outputType       string
	projectID        string
}

// WithClientData returns a copy of the custom dimensions struct with client-specific fields overwritten
func (d *CustomDimensions) WithClientData(projectNameSpace, output string) *CustomDimensions {
	d.userIDLock.Lock()
	defer d.userIDLock.Unlock()

	return &CustomDimensions{
		version:          d.version,
		branchName:       d.branchName,
		osName:           d.osName,
		osVersion:        d.osVersion,
		installSource:    d.installSource,
		machineID:        d.machineID,
		uniqID:           d.uniqID,
		sessionToken:     d.sessionToken,
		updateTag:        d.updateTag,
		userID:           d.userID,
		projectNameSpace: projectNameSpace,
		outputType:       output,
	}
}

// SetUserID is a synchronized update of the userID field.
func (d *CustomDimensions) SetUserID(userID string) {
	d.userIDLock.Lock()
	defer d.userIDLock.Unlock()

	d.userID = userID
}

func (d *CustomDimensions) toMap() map[string]string {
	d.userIDLock.Lock()
	defer d.userIDLock.Unlock()

	return map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprected)"
		"2":  d.version,
		"3":  d.branchName,
		"4":  d.userID,
		"5":  d.outputType,
		"6":  d.osName,
		"7":  d.osVersion,
		"8":  d.installSource,
		"9":  d.machineID,
		"10": d.projectNameSpace,
		"11": d.sessionToken,
		"12": d.uniqID,
		"13": d.updateTag,
		"14": d.projectID,
	}
}
