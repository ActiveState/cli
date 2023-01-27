package model

type RuntimeUsageResponse struct {
	Usage []RuntimeUsage `json:"organizations_runtime_usage"`
}

type RuntimeUsage struct {
	// ActiveDynamicRuntimes is the total number of dynamic runtimes in use
	ActiveDynamicRuntimes float64 `json:"active_runtimes"`

	// LimitDynamicRuntimes is the total number of dynamic runtimes that can be used
	LimitDynamicRuntimes float64 `json:"limit_runtimes"`
}
