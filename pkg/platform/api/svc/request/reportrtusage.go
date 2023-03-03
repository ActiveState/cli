package request

import "github.com/ActiveState/cli/internal/gqlclient"

type ReportRuntimeUsage struct {
	gqlclient.RequestBase
	pid            int
	exec           string
	dimensionsJson string
}

func NewReportRuntimeUsage(pid int, exec string, dimensionsJson string) *ReportRuntimeUsage {
	return &ReportRuntimeUsage{
		pid:            pid,
		exec:           exec,
		dimensionsJson: dimensionsJson,
	}
}

func (e *ReportRuntimeUsage) Query() string {
	return `query($pid: Int!, $exec: String!, $dimensionsJson: String!) {
		reportRuntimeUsage(pid: $pid, exec: $exec, dimensionsJson: $dimensionsJson) {
			received
		}
	}`
}

func (e *ReportRuntimeUsage) Vars() map[string]interface{} {
	return map[string]interface{}{
		"pid":            e.pid,
		"exec":           e.exec,
		"dimensionsJson": e.dimensionsJson,
	}
}
