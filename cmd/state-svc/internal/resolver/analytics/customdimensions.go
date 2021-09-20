package analytics

type customDimensions struct {
	version       string
	branchName    string
	userID        string
	osName        string
	osVersion     string
	installSource string
	machineID     string
	uniqID        string
	sessionToken  string
	updateTag     string
}

func (d *customDimensions) toMap(projectName, output, userID string) map[string]string {
	return map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprected)"
		"2":  d.version,
		"3":  d.branchName,
		"4":  userID,
		"5":  output,
		"6":  d.osName,
		"7":  d.osVersion,
		"8":  d.installSource,
		"9":  d.machineID,
		"10": projectName,
		"11": d.sessionToken,
		"12": d.uniqID,
		"13": d.updateTag,
	}
}
