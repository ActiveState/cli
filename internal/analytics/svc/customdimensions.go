package svc

type CustomDimensions struct {
	version          string
	branchName       string
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
func (d *CustomDimensions) WithClientData(projectNameSpace, output, userID string) *CustomDimensions {
	res := *d
	res.projectNameSpace = projectNameSpace
	res.outputType = output
	res.userID = userID
	return &res
}

func (d *CustomDimensions) toMap() map[string]string {
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
