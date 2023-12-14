//go:build windows
// +build windows

package e2e

const (
	testPath             = `C:\Windows\system32;C:\Windows;C:\Windows\System32\Wbem;C:\Windows\System32\WindowsPowerShell\v1.0\;C:\Program Files\PowerShell\7\`
	systemHomeEnvVarName = "USERPROFILE"
)
