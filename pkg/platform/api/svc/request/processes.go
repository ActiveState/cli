package request

type GetProcessesInUse struct {
	execDir string
}

func NewGetProcessesInUse(execDir string) *GetProcessesInUse {
	return &GetProcessesInUse{execDir}
}

func (c *GetProcessesInUse) Query() string {
	return `query($execDir: String!) {
		getProcessesInUse(execDir: $execDir) {
			exe
			pid
		}
	}`
}

func (c *GetProcessesInUse) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{"execDir": c.execDir}, nil
}
