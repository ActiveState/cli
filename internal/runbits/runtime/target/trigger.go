package target

import (
	"fmt"
	"strings"
)

type Trigger string

func (t Trigger) String() string {
	return string(t)
}

const (
	TriggerActivate           Trigger = "activate"
	TriggerScript             Trigger = "script"
	TriggerDeploy             Trigger = "deploy"
	TriggerExec               Trigger = "exec-cmd"
	TriggerExecutor           Trigger = "exec"
	TriggerResetExec          Trigger = "reset-exec"
	TriggerSwitch             Trigger = "switch"
	TriggerImport             Trigger = "import"
	TriggerInit               Trigger = "init"
	TriggerPackage            Trigger = "package"
	TriggerLanguage           Trigger = "language"
	TriggerPlatform           Trigger = "platform"
	TriggerManifest           Trigger = "manifest"
	TriggerPull               Trigger = "pull"
	TriggerRefresh            Trigger = "refresh"
	TriggerReset              Trigger = "reset"
	TriggerRevert             Trigger = "revert"
	TriggerOffline            Trigger = "offline"
	TriggerShell              Trigger = "shell"
	TriggerCheckout           Trigger = "checkout"
	TriggerCommit             Trigger = "commit"
	TriggerUse                Trigger = "use"
	TriggerOfflineInstaller   Trigger = "offline-installer"
	TriggerOfflineUninstaller Trigger = "offline-uninstaller"
	TriggerBuilds             Trigger = "builds"
	triggerUnknown            Trigger = "unknown"
)

func NewExecTrigger(cmd string) Trigger {
	return Trigger(fmt.Sprintf("%s: %s", TriggerExec, cmd))
}

func (t Trigger) IndicatesUsage() bool {
	// All triggers should indicate runtime use except for refreshing executors
	return !strings.EqualFold(string(t), string(TriggerResetExec))
}
