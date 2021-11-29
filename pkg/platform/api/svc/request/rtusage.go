package request

type RuntimeUsage struct {
	pid            int
	exec           string
	dimensionsJson string
}

func NewRuntimeUsage(pid int, exec string, dimensionsJson string) *RuntimeUsage {
	return &RuntimeUsage{
		pid:            pid,
		exec:           exec,
		dimensionsJson: dimensionsJson,
	}
}

func (e *RuntimeUsage) Query() string {
	return `query($pid: Int!, $exec: String!, $dimensionsJson: String!) {
		runtimeUsage(pid: $pid, exec: $exec, dimensionsJson: $dimensionsJson) {
			received
		}
	}`
}

func (e *RuntimeUsage) Vars() map[string]interface{} {
	return map[string]interface{}{
		"pid":            e.pid,
		"exec":           e.exec,
		"dimensionsJson": e.dimensionsJson,
	}
}
