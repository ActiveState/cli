package request

type CheckRuntimeLastUsed struct{}

func NewCheckRuntimeLastUsed() *CheckRuntimeLastUsed {
	return &CheckRuntimeLastUsed{}
}

func (e *CheckRuntimeLastUsed) Query() string {
	return `query() {
		checkRuntimeLastUsed() {
			times
		}
	}`
}

func (e *CheckRuntimeLastUsed) Vars() map[string]interface{} {
	return map[string]interface{}{}
}
