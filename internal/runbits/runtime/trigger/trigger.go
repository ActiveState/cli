package trigger

import (
	"fmt"
)

type Trigger string

func (t Trigger) String() string {
	return string(t)
}

const (
	TriggerActivate Trigger = "activate"
	TriggerScript   Trigger = "script"
	TriggerDeploy   Trigger = "deploy"
	TriggerExec     Trigger = "exec-cmd"
	TriggerExecutor Trigger = "exec"
	TriggerSwitch   Trigger = "switch"
	TriggerImport   Trigger = "import"
	TriggerInit     Trigger = "init"
	TriggerPackage  Trigger = "package"
	TriggerLanguage Trigger = "language"
	TriggerPlatform Trigger = "platform"
	TriggerPull     Trigger = "pull"
	TriggerRefresh  Trigger = "refresh"
	TriggerReset    Trigger = "reset"
	TriggerRevert   Trigger = "revert"
	TriggerShell    Trigger = "shell"
	TriggerCheckout Trigger = "checkout"
	TriggerUse      Trigger = "use"
	TriggerInstall  Trigger = "install"
)

func NewExecTrigger(cmd string) Trigger {
	return Trigger(fmt.Sprintf("%s: %s", TriggerExec, cmd))
}
