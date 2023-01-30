package request

type CheckRuntimeUsage struct {
	organizationName string
}

func NewCheckRuntimeUsage(organizationName string) *CheckRuntimeUsage {
	return &CheckRuntimeUsage{
		organizationName: organizationName,
	}
}

func (e *CheckRuntimeUsage) Query() string {
	return `query($organizationName: String!) {
		checkRuntimeUsage(organizationName: $organizationName) {
			limit
			usage
		}
	}`
}

func (e *CheckRuntimeUsage) Vars() map[string]interface{} {
	return map[string]interface{}{
		"organizationName": e.organizationName,
	}
}
