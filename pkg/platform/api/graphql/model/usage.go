package model

type RuntimeUsageResponse struct {
	Usage []RuntimeUsage `json:"organizations_runtime_usage"`
}

type RuntimeUsage struct {
	OrganizationID string `json:"organization_id"`
	Week           Date   `json:"week_of"`

	// ActiveRuntimesDynamicAndStatic are the total number of runtimes in use regardless of whether they are static or dynamic
	ActiveRuntimesDynamicAndStatic float64 `json:"total_runtimes"`

	// ActiveStaticRuntimesFromDeploy are the total number of static runtimes in use that were installed with `state deploy`
	ActiveStaticRuntimesFromDeploy float64 `json:"static_runtimes_state_deploy"`

	// ActiveStaticRuntimesFromInstaller are the total number of static runtimes in use that were installed with the installer
	ActiveStaticRuntimesFromInstaller float64 `json:"static_runtimes_next_gen_installer"`

	// ActiveDynamicRuntimes is the total number of dynamic runtimes in use
	ActiveDynamicRuntimes float64 `json:"active_runtimes"`

	// LimitDynamicRuntimes is the total number of dynamic runtimes that can be used
	LimitDynamicRuntimes float64 `json:"limit_runtimes"`
}
