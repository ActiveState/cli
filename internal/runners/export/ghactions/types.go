package ghactions

type Workflow struct {
	Name string                 `yaml:"name,omitempty"`
	On   WorkflowTrigger        `yaml:"on"`
	Jobs map[string]WorkflowJob `yaml:"jobs"`
}

type WorkflowTrigger struct {
	Push        WorkflowBranches `yaml:"push"`
	PullRequest WorkflowBranches `yaml:"pull_request"`
}

type WorkflowBranches struct {
	Branches []string `yaml:"branches"`
}

type WorkflowJob struct {
	Env    map[string]interface{} `yaml:"env,omitempty"`
	RunsOn string                 `yaml:"runs-on,omitempty"`
	Steps  []WorkflowStep         `yaml:"steps,omitempty"`
}

type WorkflowStep struct {
	Name  string `yaml:"name,omitempty"`
	Uses  string `yaml:"uses,omitempty"`
	Run   string `yaml:"run,omitempty"`
	Shell string `yaml:"shell,omitempty"`
}
