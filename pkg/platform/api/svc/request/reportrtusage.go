package request

type ReportRuntimeUsage struct {
	pid            int
	exec           string
	source         string
	dimensionsJson string
}

func NewReportRuntimeUsage(pid int, exec, source string, dimensionsJson string) *ReportRuntimeUsage {
	return &ReportRuntimeUsage{
		pid:            pid,
		exec:           exec,
		source:         source,
		dimensionsJson: dimensionsJson,
	}
}

func (e *ReportRuntimeUsage) Query() string {
	return `query($pid: Int!, $exec: String!, $source: String!, $dimensionsJson: String!) {
		reportRuntimeUsage(pid: $pid, exec: $exec, source: $source, dimensionsJson: $dimensionsJson) {
			received
		}
	}`
}

func (e *ReportRuntimeUsage) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{
		"pid":            e.pid,
		"exec":           e.exec,
		"source":         e.source,
		"dimensionsJson": e.dimensionsJson,
	}, nil
}
